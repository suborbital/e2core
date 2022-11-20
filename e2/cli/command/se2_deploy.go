package command

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/builder/template"
	"github.com/suborbital/e2core/e2/cli/input"
	"github.com/suborbital/e2core/e2/cli/localproxy"
	"github.com/suborbital/e2core/e2/cli/release"
	"github.com/suborbital/e2core/e2/cli/repl"
	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

type deployData struct {
	SCCVersion       string
	EnvToken         string
	BuilderDomain    string
	StorageClassName string
}

const proxyDefaultPort int = 80

// SE2DeployCommand returns the SE2 deploy command.
func SE2DeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy SE2",
		Long:  `Deploy Suborbital Extension Engine (SE2) using Kubernetes or Docker Compose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			localInstall := cmd.Flags().Changed(localFlag)
			shouldReset := cmd.Flags().Changed(resetFlag)
			branch, _ := cmd.Flags().GetString(branchFlag)
			tag, _ := cmd.Flags().GetString(versionFlag)

			if !localInstall {
				if err := introAcceptance(); err != nil {
					return err
				}
			}

			proxyPort, _ := cmd.Flags().GetInt(proxyPortFlag)
			if proxyPort < 1 || proxyPort > (2<<16)-1 {
				return errors.New("🚫 proxy-port must be between 1 and 65535")
			}

			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to Getwd")
			}

			bctx, err := project.ForDirectory(cwd)
			if err != nil {
				return errors.Wrap(err, "🚫 failed to project.ForDirectory")
			}

			// if the --reset flag was passed or there's no existing manifests
			// then we need to 'build the world' from scratch.
			if shouldReset || !manifestsExist(bctx) {
				util.LogStart("preparing deployment")

				// if there are any existing deployment manifests sitting around, let's replace them.
				if err := removeExistingManifests(bctx); err != nil {
					return errors.Wrap(err, "failed to removeExistingManifests")
				}

				_, err = util.Mkdir(bctx.Cwd, ".suborbital")
				if err != nil {
					return errors.Wrap(err, "🚫 failed to Mkdir")
				}

				templatesPath, err := template.TemplatesExist(defaultRepo, branch)
				if err != nil {
					templatesPath, err = template.UpdateTemplates(defaultRepo, branch)
					if err != nil {
						return errors.Wrap(err, "🚫 failed to UpdateTemplates")
					}
				}

				envToken, err := getEnvToken()
				if err != nil {
					return errors.Wrap(err, "🚫 failed to getEnvToken")
				}

				data := deployData{
					SCCVersion: tag,
					EnvToken:   envToken,
				}

				templateName := "scc-docker"

				if !localInstall {
					data.BuilderDomain, err = getBuilderDomain()
					if err != nil {
						return errors.Wrap(err, "🚫 failed to getBuilderDomain")
					}

					data.StorageClassName, err = getStorageClass()
					if err != nil {
						return errors.Wrap(err, "🚫 failed to getStorageClass")
					}

					templateName = "scc-k8s"
				}

				if err := template.ExecTmplDir(bctx.Cwd, "", templatesPath, templateName, data); err != nil {
					return errors.Wrap(err, "🚫 failed to ExecTmplDir")
				}

				util.LogDone("ready to start installation")
			}

			dryRun, _ := cmd.Flags().GetBool(dryRunFlag)

			if dryRun {
				util.LogInfo("aborting due to dry-run, manifest files remain in " + bctx.Cwd)
				return nil
			}

			util.LogStart("installing...")

			if localInstall {
				var compose string
				if _, err := util.Command.Run("docker compose version 2>&1 >/dev/null"); err == nil {
					// Use Compose v2 if we're positive we have it
					compose = "docker compose"
				} else if _, err := exec.LookPath("docker-compose"); err == nil {
					// Fall back to legacy compose if available.
					compose = "docker-compose"
				} else {
					// YOLO. Try Compose V2 anyway. Works with containerd/nerdctl.
					// See: https://github.com/containerd/nerdctl/issues/1368
					compose = "docker compose"
				}

				command := fmt.Sprintf("%s up -d", compose)

				if _, err := util.Command.Run(command); err != nil {
					util.LogInfo("Is Docker Compose installed? https://docs.docker.com/compose/install/")
					return errors.Wrapf(err, "🚫 failed to run `%s`", command)
				}

				util.LogInfo(fmt.Sprintf("use `docker ps` and `%s logs` to check deployment status", compose))

				proxyPortStr := strconv.Itoa(proxyPort)
				proxy := localproxy.New("editor.suborbital.network", proxyPortStr)

				go func() {
					if err := proxy.Start(); err != nil {
						log.Fatal(err)
					}
				}()

				// this is to give the proxy server some room to bind to the port and start up
				// it's not ideal, but the least gross way to ensure a good experience.
				time.Sleep(time.Second * 1)

				repl := repl.New(proxyPortStr)
				repl.Run()

			} else {
				if _, err := util.Command.Run("kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.4.0/keda-2.4.0.yaml"); err != nil {
					return errors.Wrap(err, "🚫 failed to install KEDA")
				}

				// we don't care if this fails, so don't check error.
				util.Command.Run("kubectl create ns suborbital")

				if err := createConfigMap(cwd); err != nil {
					return errors.Wrap(err, "failed to createConfigMap")
				}

				if _, err := util.Command.Run("kubectl apply -f .suborbital/"); err != nil {
					return errors.Wrap(err, "🚫 failed to kubectl apply")
				}

				util.LogInfo("use `kubectl get pods -n suborbital` and `kubectl get svc -n suborbital` to check deployment status")
			}

			util.LogDone("installation complete!")

			return nil
		},
	}

	cmd.Flags().String(branchFlag, defaultBranch, "git branch to download templates from")
	cmd.Flags().String(versionFlag, release.SCCTag, "Docker tag to use for control plane images")
	cmd.Flags().Int(proxyPortFlag, proxyDefaultPort, "port that the Editor proxy listens on")
	cmd.Flags().Bool(localFlag, false, "deploy locally using Docker Compose")
	cmd.Flags().Bool(dryRunFlag, false, "prepare the deployment in the .suborbital directory, but do not apply it")
	cmd.Flags().Bool(resetFlag, false, "reset the deployment to default (replaces docker-compose.yaml and/or Kubernetes manifests)")

	return cmd
}

func introAcceptance() error {
	fmt.Print(`
Suborbital Extension Engine (SE2) Installer

BEFORE YOU CONTINUE:
	- You must first run "e2 se2 create token <email>" to get an environment token

	- You must have kubectl installed in PATH, and it must be connected to the cluster you'd like to use

	- You must be able to set up DNS records for the builder service after this installation completes
			- Choose the DNS name you'd like to use before continuing, e.g. builder.acmeco.com

	- Subo will attempt to determine the default storage class for your Kubernetes cluster,
	  but if is unable to do so you will need to provide one
			- See the SE2 documentation for more details

	- Subo will install the KEDA autoscaler into your cluster. It will not affect any existing deployments.

Are you ready to continue? (y/N): `)

	answer, err := input.ReadStdinString()
	if err != nil {
		return errors.Wrap(err, "failed to ReadStdinString")
	}

	if !strings.EqualFold(answer, "y") {
		return errors.New("aborting")
	}

	return nil
}

// getEnvToken gets the environment token from stdin.
func getEnvToken() (string, error) {
	existing, err := util.ReadEnvironmentToken()
	if err == nil {
		util.LogInfo("using cached environment token")
		return existing, nil
	}

	fmt.Print("Enter your environment token: ")
	token, err := input.ReadStdinString()

	if err != nil {
		return "", errors.Wrap(err, "failed to ReadStdinString")
	}

	if len(token) != 32 {
		return "", errors.New("token must be 32 characters in length")
	}

	if err := util.WriteEnvironmentToken(token); err != nil {
		util.LogWarn(err.Error())
		return token, nil

	} else {
		util.LogInfo("saved environment token to cache")
	}

	return token, nil
}

// getBuilderDomain gets the environment token from stdin.
func getBuilderDomain() (string, error) {
	fmt.Print("Enter the domain name that will be used for the builder service: ")
	domain, err := input.ReadStdinString()
	if err != nil {
		return "", errors.Wrap(err, "failed to ReadStdinString")
	}

	if len(domain) == 0 {
		return "", errors.New("domain must not be empty")
	}

	return domain, nil
}

// getStorageClass gets the storage class to use.
func getStorageClass() (string, error) {
	defaultClass, err := detectStorageClass()
	if err != nil {
		// that's fine, continue.
		fmt.Println("failed to automatically detect Kubernetes storage class:", err.Error())
	} else if defaultClass != "" {
		fmt.Println("using default storage class: ", defaultClass)
		return defaultClass, nil
	}

	fmt.Print("Enter the Kubernetes storage class to use: ")
	storageClass, err := input.ReadStdinString()
	if err != nil {
		return "", errors.Wrap(err, "failed to ReadStdinString")
	}

	if len(storageClass) == 0 {
		return "", errors.New("storage class must not be empty")
	}

	return storageClass, nil
}

func detectStorageClass() (string, error) {
	output, err := util.Command.Run("kubectl get storageclass --output=name")
	if err != nil {
		return "", errors.Wrap(err, "failed to get default storageclass")
	}

	// output will look like: storageclass.storage.k8s.io/do-block-storage
	// so split on the / and return the last part.

	outputParts := strings.Split(output, "/")
	if len(outputParts) != 2 {
		return "", errors.New("could not automatically determine storage class")
	}

	return outputParts[1], nil
}

func createConfigMap(cwd string) error {
	configFilepath := filepath.Join(cwd, "config", "scc-config.yaml")

	_, err := os.Stat(configFilepath)
	if err != nil {
		return errors.Wrap(err, "failed to Stat scc-config.yaml")
	}

	if _, err := util.Command.Run(fmt.Sprintf("kubectl create configmap scc-config --from-file=scc-config.yaml=%s -n suborbital", configFilepath)); err != nil {
		return errors.Wrap(err, "failed to create configmap (you may need to run `kubectl delete configmap scc-config -n suborbital`)")
	}

	return nil
}

func manifestsExist(bctx *project.Context) bool {
	if _, err := os.Stat(filepath.Join(bctx.Cwd, ".suborbital")); err == nil {
		return true
	}

	if _, err := os.Stat(filepath.Join(bctx.Cwd, "docker-compose.yml")); err == nil {
		return true
	}

	return false
}

func removeExistingManifests(bctx *project.Context) error {
	// start with a clean slate.
	if _, err := os.Stat(filepath.Join(bctx.Cwd, ".suborbital")); err == nil {
		if err := os.RemoveAll(filepath.Join(bctx.Cwd, ".suborbital")); err != nil {
			return errors.Wrap(err, "failed to RemoveAll .suborbital")
		}
	}

	if _, err := os.Stat(filepath.Join(bctx.Cwd, "docker-compose.yml")); err == nil {
		if err := os.Remove(filepath.Join(bctx.Cwd, "docker-compose.yml")); err != nil {
			return errors.Wrap(err, "failed to Remove docker-compose.yml")
		}
	}

	return nil
}

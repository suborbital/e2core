package command

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/templater"
	"github.com/suborbital/e2core/e2/util"
	"github.com/suborbital/e2core/e2core/release"
)

type deployData struct {
	E2CoreTag        string
	EnvToken         string
	BuilderDomain    string
	StorageClassName string
}

const defaultRepo string = "suborbital/e2core"
const defaultBranch = "v" + release.E2CoreServerDotVersion

// DeployCommand returns the SE2 deploy command.
func DeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy E2 Core to Kubernetes",
		Long:  "Deploy E2 Core to Kubernetes",
		RunE: func(cmd *cobra.Command, args []string) error {
			shouldReset := cmd.Flags().Changed(resetFlag)
			repo, _ := cmd.Flags().GetString(repoFlag)
			branch, _ := cmd.Flags().GetString(branchFlag)
			forceUpdateTemplates, _ := cmd.Flags().GetBool(updateTemplatesFlag)

			if err := introAcceptance(); err != nil {
				return err
			}

			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "🚫 failed to Getwd")
			}

			workingDirectory, err := filepath.Abs(cwd)
			if err != nil {
				return errors.Wrap(err, "🚫 failed to get absolute path")
			}

			if forceUpdateTemplates {
				util.LogInfo(fmt.Sprintf("updating (forced) templates from %s (%s)", repo, branch))

				_, err = templater.UpdateTemplates(repo, branch)
				if err != nil {
					return errors.Wrap(err, "🚫 failed to UpdateTemplates")
				}
			}

			templatesPath, err := templater.TemplatesExist(repo, branch)
			// Fetch templates if they don't exist
			if err != nil {
				util.LogInfo(fmt.Sprintf("updating templates from %s (%s)", repo, branch))
				templatesPath, err = templater.UpdateTemplates(repo, branch)

				if err != nil {
					return errors.Wrap(err, "🚫 failed to UpdateTemplates")
				}
			}

			// if the --reset flag was passed or there's no existing manifests
			// then we need to 'build the world' from scratch.
			if shouldReset || !manifestsExist(workingDirectory) {
				util.LogStart("preparing deployment")

				// if there are any existing deployment manifests sitting around, let's replace them.
				if err := removeExistingManifests(workingDirectory); err != nil {
					return errors.Wrap(err, "🚫 failed to removeExistingManifests")
				}

				_, err = util.Mkdir(workingDirectory, ".suborbital")
				if err != nil {
					return errors.Wrap(err, "🚫 failed to Mkdir")
				}

				envToken, err := getEnvToken()
				if err != nil {
					return errors.Wrap(err, "🚫 failed to getEnvToken")
				}

				data := deployData{
					E2CoreTag: "v" + release.E2CoreServerDotVersion,
					EnvToken:  envToken,
				}

				templateName := "e2core-k8s"

				data.StorageClassName, err = getStorageClass()
				if err != nil {
					return errors.Wrap(err, "🚫 failed to getStorageClass")
				}

				if err := templater.ExecTmplDir(workingDirectory, "", templatesPath, templateName, data); err != nil {
					return errors.Wrap(err, "🚫 failed to ExecTmplDir")
				}

				util.LogDone("ready to start installation")
			}

			dryRun, _ := cmd.Flags().GetBool(dryRunFlag)

			if dryRun {
				util.LogInfo("aborting due to dry-run, manifest files remain in " + workingDirectory)
				return nil
			}

			util.LogStart("installing...")

			if _, err := util.Command.Run("kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.4.0/keda-2.4.0.yaml"); err != nil {
				return errors.Wrap(err, "🚫 failed to install KEDA")
			}

			// we don't care if this fails, so don't check error.
			util.Command.Run("kubectl create ns suborbital")

			if err := createConfigMap(cwd); err != nil {
				return errors.Wrap(err, "🚫 failed to createConfigMap")
			}

			if _, err := util.Command.Run("kubectl apply -f .suborbital/"); err != nil {
				return errors.Wrap(err, "🚫 failed to kubectl apply")
			}

			util.LogDone("installation complete!")

			return nil
		},
	}

	cmd.Flags().String(repoFlag, defaultRepo, "git repo to download templates from")
	cmd.Flags().String(branchFlag, defaultBranch, "git branch to download templates from")
	cmd.Flags().Bool(dryRunFlag, false, "prepare the deployment in the .suborbital directory, but do not apply it")
	cmd.Flags().Bool(resetFlag, false, "reset the deployment to default (replaces Kubernetes manifests)")
	cmd.Flags().Bool(updateTemplatesFlag, false, "forces templates to be updated")

	return cmd
}

// TODO: update this
func introAcceptance() error {
	fmt.Print(`
Suborbital Extension Engine (SE2) Installer

BEFORE YOU CONTINUE:
	- You must first run "subo se2 create token <email>" to get an environment token

	- You must have kubectl installed in PATH, and it must be connected to the cluster you'd like to use

	- You must be able to set up DNS records for the builder service after this installation completes
			- Choose the DNS name you'd like to use before continuing, e.g. builder.acmeco.com

	- Subo will attempt to determine the default storage class for your Kubernetes cluster,
	  but if is unable to do so you will need to provide one
			- See the SE2 documentation for more details

	- Subo will install the KEDA autoscaler into your cluster. It will not affect any existing deployments.

Are you ready to continue? (y/N): `)

	answer, err := util.ReadStdinString()
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
	token, err := util.ReadStdinString()

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
	storageClass, err := util.ReadStdinString()
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
	configFilepath := filepath.Join(cwd, "config", "se2-config.yaml")

	_, err := os.Stat(configFilepath)
	if err != nil {
		return errors.Wrap(err, "failed to Stat se2-config.yaml")
	}

	if _, err := util.Command.Run(fmt.Sprintf("kubectl create configmap se2-config --from-file=se2-config.yaml=%s -n suborbital", configFilepath)); err != nil {
		return errors.Wrap(err, "failed to create configmap (you may need to run `kubectl delete configmap se2-config -n suborbital`)")
	}

	return nil
}

func manifestsExist(workingDirectory string) bool {
	if _, err := os.Stat(filepath.Join(workingDirectory, ".suborbital")); err == nil {
		return true
	}

	return false
}

func removeExistingManifests(workingDirectory string) error {
	// start with a clean slate.
	if _, err := os.Stat(filepath.Join(workingDirectory, ".suborbital")); err == nil {
		if err := os.RemoveAll(filepath.Join(workingDirectory, ".suborbital")); err != nil {
			return errors.Wrap(err, "failed to RemoveAll .suborbital")
		}
	}

	return nil
}

package deployer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/builder/template"
	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

const (
	k8sDeployJobType = "kubernetes"
)

// K8sDeployJob represents a deployment job.
type K8sDeployJob struct {
	repo            string
	branch          string
	domain          string
	updateTemplates bool
}

type deploymentData struct {
	Identifier string
	Version    int64
	ImageName  string
	Domain     string
}

// NewK8sDeployJob creates a new deploy job.
func NewK8sDeployJob(repo, branch, domain string, updateTemplates bool) DeployJob {
	k := &K8sDeployJob{
		repo:            repo,
		branch:          branch,
		domain:          domain,
		updateTemplates: updateTemplates,
	}

	return k
}

// Typw returns the deploy job typw.
func (k *K8sDeployJob) Type() string {
	return k8sDeployJobType
}

// Deploy executes the deployment.
func (k *K8sDeployJob) Deploy(log util.FriendlyLogger, ctx *project.Context) error {
	imageName, err := project.DockerNameFromConfig(ctx.TenantConfig)
	if err != nil {
		return errors.Wrap(err, "failed to DockerNameFromDirective")
	}

	data := deploymentData{
		Identifier: strings.Replace(ctx.TenantConfig.Identifier, ".", "-", -1),
		Version:    ctx.TenantConfig.TenantVersion,
		ImageName:  imageName,
		Domain:     k.domain,
	}

	if err := os.RemoveAll(filepath.Join(ctx.Cwd, ".deployment")); err != nil {
		if !os.IsNotExist(err) {
			return errors.Wrap(err, "failed to RemoveAll deployment files")
		}
	}

	if err := os.MkdirAll(filepath.Join(ctx.Cwd, ".deployment"), util.PermDirectory); err != nil {
		return errors.Wrap(err, "failed to MkdirAll .deployment")
	}

	templatesPath, err := template.FullPath(k.repo, k.branch)
	if err != nil {
		return errors.Wrap(err, "failed to template.FullPath")
	}

	if k.updateTemplates {
		templatesPath, err = template.UpdateTemplates(k.repo, k.branch)
		if err != nil {
			return errors.Wrap(err, "🚫 failed to UpdateTemplates")
		}
	}

	if err := template.ExecTmplDir(ctx.Cwd, ".deployment", templatesPath, "k8s", data); err != nil {
		// if the templates are missing, try updating them and exec again.
		if err == template.ErrTemplateMissing {
			templatesPath, err = template.UpdateTemplates(k.repo, k.branch)
			if err != nil {
				return errors.Wrap(err, "🚫 failed to UpdateTemplates")
			}

			if err := template.ExecTmplDir(ctx.Cwd, ".deployment", templatesPath, "k8s", data); err != nil {
				return errors.Wrap(err, "🚫 failed to ExecTmplDir")
			}
		} else {
			return errors.Wrap(err, "🚫 failed to ExecTmplDir")
		}
	}

	if out, err := util.Command.Run("kubectl create ns suborbital"); err != nil {
		log.LogWarn(fmt.Sprintf("failed to create `suborbital` namespace (may alrady exist): %s", out))
	}

	if _, err := util.Command.Run("kubectl apply -f .deployment/"); err != nil {
		return errors.Wrap(err, "failed to Run kubectl apply")
	}

	return nil
}

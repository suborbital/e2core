package command

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/builder/template"
	"github.com/suborbital/e2core/e2/cli/release"
	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

const (
	defaultRepo   = "suborbital/templates"
	defaultBranch = "main"
)

type projectData struct {
	Name           string
	Environment    string
	APIVersion     string
	RuntimeVersion string
}

// CreateProjectCmd returns the build command.
func CreateProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project <name>",
		Short: "Create a new project",
		Long:  `Create a new project for E2Core or Suborbital Extension Engine (SE2)`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to Getwd")
			}

			bctx, err := project.ForDirectory(cwd)
			if err != nil {
				return errors.Wrap(err, "🚫 failed to project.ForDirectory")
			}

			util.LogStart(fmt.Sprintf("creating project %s", name))

			path, err := util.Mkdir(bctx.Cwd, name)
			if err != nil {
				return errors.Wrap(err, "🚫 failed to Mkdir")
			}

			branch, _ := cmd.Flags().GetString(branchFlag)
			environment, _ := cmd.Flags().GetString(environmentFlag)

			templatesPath, err := template.FullPath(defaultRepo, branch)
			if err != nil {
				return errors.Wrap(err, "🚫 failed to template.FullPath")
			}

			if update, _ := cmd.Flags().GetBool(updateTemplatesFlag); update {
				templatesPath, err = template.UpdateTemplates(defaultRepo, branch)
				if err != nil {
					return errors.Wrap(err, "🚫 failed to UpdateTemplates")
				}
			}

			data := projectData{
				Name:           name,
				Environment:    environment,
				APIVersion:     release.FFIVersion,
				RuntimeVersion: release.RuntimeVersion,
			}

			if err := template.ExecTmplDir(bctx.Cwd, name, templatesPath, "project", data); err != nil {
				// if the templates are missing, try updating them and exec again.
				if err == template.ErrTemplateMissing {
					templatesPath, err = template.UpdateTemplates(defaultRepo, branch)
					if err != nil {
						return errors.Wrap(err, "🚫 failed to UpdateTemplates")
					}

					if err := template.ExecTmplDir(bctx.Cwd, name, templatesPath, "project", data); err != nil {
						return errors.Wrap(err, "🚫 failed to ExecTmplDir")
					}
				} else {
					return errors.Wrap(err, "🚫 failed to ExecTmplDir")
				}
			}

			util.LogDone(path)

			if _, err := util.Command.Run(fmt.Sprintf("git init ./%s", name)); err != nil {
				return errors.Wrap(err, "🚫 failed to initialize Run git init")
			}

			return nil
		},
	}

	cmd.Flags().String(branchFlag, defaultBranch, "git branch to download templates from")
	cmd.Flags().String(environmentFlag, "com.suborbital", "project environment name (your company's reverse domain")
	cmd.Flags().Bool(updateTemplatesFlag, false, "update with the newest templates")

	return cmd
}

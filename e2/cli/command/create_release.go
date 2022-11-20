package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v2"

	"github.com/suborbital/e2core/e2/cli/util"
)

// DotSuboFile describes a .e2 file for controlling releases.
type DotSuboFile struct {
	DotVersionFiles []string `yaml:"dotVersionFiles"`
	PreMakeTargets  []string `yaml:"preMakeTargets"`
	PostMakeTargets []string `yaml:"postMakeTargets"`
}

// CreateReleaseCmd returns the create release command
// this is only available for development builds.
func CreateReleaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release <version> <title>",
		Short: "Create a new release",
		Long:  `Tag a new version and create a new GitHub release, configured using the .e2.yml file.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			util.LogStart("checking release conditions")
			cwd, _ := cmd.Flags().GetString("dir")

			newVersion := args[0]
			releaseName := args[1]

			// ensure the version entered is sane.
			if err := validateVersion(newVersion); err != nil {
				return errors.Wrap(err, "failed to validateVersion")
			}

			// ensure the git repo is clean, no untracked or uncommitted changes.
			if err := checkGitCleanliness(); err != nil {
				return errors.Wrap(err, "failed to checkGitCleanliness")
			}

			// ensure the current git branch is an rc branch.
			branch, err := ensureCorrectGitBranch(newVersion)
			if err != nil {
				return errors.Wrap(err, "failed to ensureCorrectGitBranch")
			}

			// ensure a .e2.yml file is present and valid.
			dotSubo, err := findDotSubo(cwd)
			if err != nil {
				return errors.Wrap(err, "failed to findDotSubo")
			} else if dotSubo == nil {
				return errors.New(".e2.yml file is missing")
			}

			// ensure a changelog exists for the release.
			changelogFilePath := filepath.Join(cwd, "changelogs", fmt.Sprintf("%s.md", newVersion))

			if err := checkChangelogFileExists(changelogFilePath); err != nil {
				return errors.Wrap(err, "failed to checkChangelogFileExists")
			}

			// ensure each of the versionFiles contains the string of the new version.
			for _, f := range dotSubo.DotVersionFiles {
				filePath := filepath.Join(cwd, f)

				if err := util.CheckFileForVersionString(filePath, newVersion); err != nil {
					if errors.Is(err, util.ErrVersionNotPresent) {
						return fmt.Errorf("required dotVersionFile %s does not contain the release version number %s", filePath, newVersion)
					}

					return errors.Wrap(err, "failed to CheckFileForVersionString")
				}
			}

			util.LogDone("release is ready to go")
			util.LogStart("running pre-make targets")

			// run all of the pre-release make targets.
			for _, target := range dotSubo.PreMakeTargets {
				targetWithVersion := strings.Replace(target, "{{ .Version }}", newVersion, -1)

				if _, err := util.Command.Run(fmt.Sprintf("make %s", targetWithVersion)); err != nil {
					return errors.Wrapf(err, "failed to run preMakeTarget %s", target)
				}
			}

			util.LogDone("pre-make targets complete")

			if shouldDryRun, _ := cmd.Flags().GetBool(dryRunFlag); shouldDryRun {
				util.LogDone("release conditions verified, terminating for dry run")
				return nil
			}

			util.LogStart("creating release")

			// ensure the local changes are pushed, create the release, and then pull down the new tag.
			if _, err := util.Command.Run("git push"); err != nil {
				return errors.Wrap(err, "failed to Run git push")
			}

			ghCommand := fmt.Sprintf("gh release create %s --title=%s --target=%s --notes-file=%s", newVersion, releaseName, branch, changelogFilePath)
			if preRelease, _ := cmd.Flags().GetBool(preReleaseFlag); preRelease {
				ghCommand += " --prerelease"
			}

			if _, err := util.Command.Run(ghCommand); err != nil {
				return errors.Wrap(err, "failed to Run gh command")
			}

			if _, err := util.Command.Run("git pull --tags"); err != nil {
				return errors.Wrap(err, "failed to Run git pull command")
			}

			util.LogDone("release created!")
			util.LogStart("running post-make targets")

			// run all of the post-release make targets.
			for _, target := range dotSubo.PostMakeTargets {
				targetWithVersion := strings.Replace(target, "{{ .Version }}", newVersion, -1)

				if _, err := util.Command.Run(fmt.Sprintf("make %s", targetWithVersion)); err != nil {
					return errors.Wrapf(err, "failed to run postMakeTarget %s", target)
				}
			}

			util.LogDone("post-make targets complete")

			return nil
		},
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "$HOME"
	}

	cmd.Flags().String(dirFlag, cwd, "the directory to create the release for")
	cmd.Flags().Bool(preReleaseFlag, false, "pass --prelease to mark the release as such")
	cmd.Flags().Bool(dryRunFlag, false, "pass --dryrun to run release condition checks and pre-make targets, but don't create the release")

	return cmd
}

func findDotSubo(cwd string) (*DotSuboFile, error) {
	dotSuboPath := filepath.Join(cwd, ".e2.yml")

	dotSuboBytes, err := ioutil.ReadFile(dotSuboPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to ReadFile")
	}

	dotSubo := &DotSuboFile{}
	if err := yaml.Unmarshal(dotSuboBytes, dotSubo); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal dotSubo file")
	}

	return dotSubo, nil
}

func checkChangelogFileExists(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		return errors.Wrap(err, "failed to Stat changelog file")
	}

	return nil
}

func checkGitCleanliness() error {
	if out, err := util.Command.Run("git diff-index --name-only HEAD"); err != nil {
		return errors.Wrap(err, "failed to git diff-index")
	} else if out != "" {
		return errors.New("project has modified files")
	}

	if out, err := util.Command.Run("git ls-files --exclude-standard --others"); err != nil {
		return errors.Wrap(err, "failed to git ls-files")
	} else if out != "" {
		return errors.New("project has untracked files")
	}

	return nil
}

func ensureCorrectGitBranch(version string) (string, error) {
	expectedBranch := fmt.Sprintf("rc-%s", version)

	branch, err := util.Command.Run("git branch --show-current")
	if err != nil {
		return "", errors.Wrap(err, "failed to Run git branch")
	}

	if strings.TrimSpace(branch) != expectedBranch {
		return "", errors.New("release must be created on an 'rc-*' branch, currently on " + branch + ", expected " + expectedBranch)
	}

	return strings.TrimSpace(branch), nil
}

func validateVersion(version string) error {
	if !strings.HasPrefix(version, "v") {
		return errors.New("version does not start with v")
	}

	if !semver.IsValid(version) {
		return errors.New("version is not valid semver")
	}

	return nil
}

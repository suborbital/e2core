package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/suborbital/e2core/e2/cli/util"
	"github.com/suborbital/e2core/e2/project"
)

// CleanCmd  removes all of the target/.build folders for Runnables and deletes the .wasm files.
func CleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove build folders and .wasm files",
		Long:  "Remove all of target/.build folders and deletes .wasm files",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return errors.Wrap(err, "failed to Getwd")
			}

			bctx, err := project.ForDirectory(cwd)
			if err != nil {
				return errors.Wrap(err, "failed to project.ForDirectory")
			}

			if len(bctx.Modules) == 0 {
				return errors.New("🚫 no runnables found in current directory (no .runnable yaml files found)")
			}

			util.LogStart(fmt.Sprintf("cleaning in %s", bctx.Cwd))

			for _, r := range bctx.Modules {
				// Delete target or .build folder.
				files, _ := ioutil.ReadDir(r.Fullpath)

				for _, file := range files {
					fullPath := filepath.Join(r.Fullpath, file.Name())
					if file.IsDir() {
						if file.Name() == "target" || file.Name() == ".build" {
							if rErr := os.RemoveAll(fullPath); rErr != nil {
								util.LogFail(errors.Wrap(rErr, "failed to RemoveAll").Error())
								continue
							}

							util.LogDone(fmt.Sprintf("removed %s", file.Name()))
						}
					} else {
						if strings.HasSuffix(file.Name(), ".wasm") || strings.HasSuffix(file.Name(), ".wasm.zip") {
							if err := os.Remove(fullPath); err != nil {
								util.LogInfo(errors.Wrap(err, "🚫 failed to Remove").Error())
								continue
							}

							util.LogDone(fmt.Sprintf("removed %s", file.Name()))
						}
					}
				}
			}

			util.LogDone("cleaned")
			return nil
		},
	}

	return cmd
}

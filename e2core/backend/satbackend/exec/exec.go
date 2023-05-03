package exec

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Run runs a command, outputting to terminal and returning the full output and/or error
// a channel is returned which, when sent on, will terminate the process that was started
func Run(cmd []string, env ...string) (string, context.CancelCauseFunc, error) {
	procUUID := uuid.New().String()
	uuidEnv := fmt.Sprintf("%s_UUID=%s", strings.ToUpper(cmd[0]), procUUID)
	env = append(env, uuidEnv)

	// Create a context with a cancel with cause functionality. Instead of reaping the process by killing by process id,
	// we're going to call the cancel function for this process.
	ctx, cxl := context.WithCancelCause(context.Background())

	// Set up the command. cmd[0] is e2core, 1:... is mod start <fqmn>.
	command := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	command.Env = env
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	err := command.Start()
	if err != nil {
		return "", nil, errors.Wrap(err, "command.Start()")
	}

	return procUUID, cxl, nil
}

// this is unused but we may want to do logging-to-speficig-directory some time in the
// future, so we're leaving it here.
// logfilePath returns the directory that Info files should be written to
// func logfilePath(uuid string) (string, error) {
// 	config, err := os.UserConfigDir()
// 	if err != nil {
// 		return "", errors.Wrap(err, "failed to UserConfigDir")
// 	}

// 	dir := filepath.Join(config, "suborbital", "log")

// 	if err := os.MkdirAll(dir, 0755); err != nil {
// 		return "", errors.Wrap(err, "failed to MkdirAll")
// 	}

// 	filePath := filepath.Join(dir, fmt.Sprintf("%s.log", uuid))

// 	return filePath, nil
// }

package exec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Run runs a command, outputting to terminal and returning the full output and/or error
// a channel is returned which, when sent on, will terminate the process that was started
func Run(cmd string, env ...string) (string, int, error) {
	// you can uncomment this below if you want to see exactly the commands being run
	fmt.Println("▶️", cmd)

	parts := strings.Split(cmd, " ")

	// add an environment variable with a UUID
	// if the command is `sat`, then the var will be
	// SAT_UUID=asdfghjkl
	procUUID := uuid.New().String()
	uuidEnv := fmt.Sprintf("%s_UUID=%s", strings.ToUpper(parts[0]), procUUID)
	env = append(env, uuidEnv)

	logPath, err := logfilePath(procUUID)
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to logFilePath")
	}

	logEnv := fmt.Sprintf("%s_LOG_FILE=%s", strings.ToUpper(parts[0]), logPath)
	env = append(env, logEnv)

	// augment the provided env with the env of the parent
	env = append(env, os.Environ()...)

	binPath, err := exec.LookPath(parts[0])
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to LookPath")
	}

	info := &syscall.ProcAttr{
		Env:   env,
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()},
	}

	pid, err := syscall.ForkExec(binPath, parts, info)
	if err != nil {
		return "", 0, errors.Wrap(err, "failed to ForkExec")
	}

	return procUUID, pid, nil
}

// logfilePath returns the directory that Info files should be written to
func logfilePath(uuid string) (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to UserConfigDir")
	}

	dir := filepath.Join(config, "suborbital", "log")

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to MkdirAll")
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.log", uuid))

	return filePath, nil
}

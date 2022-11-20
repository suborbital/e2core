package util

import (
	"bytes"
	"io"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

type CommandRunner interface {
	Run(cmd string) (string, error)
	RunInDir(cmd, dir string) (string, error)
}

type silentOutput bool

const (
	SilentOutput = true
	NormalOutput = false
)

type CommandLineExecutor struct {
	silent silentOutput
	writer io.Writer
}

// Command is a barebones command executor.
var Command = &CommandLineExecutor{}

// NewCommandLineExecutor creates a new CommandLineExecutor with the given configuration.
func NewCommandLineExecutor(silent silentOutput, writer io.Writer) *CommandLineExecutor {
	return &CommandLineExecutor{
		silent: silent,
		writer: writer,
	}
}

// Run runs a command, outputting to terminal and returning the full output and/or error.
func (d *CommandLineExecutor) Run(cmd string) (string, error) {
	return run(cmd, "", d.silent, d.writer)
}

// RunInDir runs a command in the specified directory and returns the full output or error.
func (d *CommandLineExecutor) RunInDir(cmd, dir string) (string, error) {
	return run(cmd, dir, d.silent, d.writer)
}

func run(cmd, dir string, silent silentOutput, writer io.Writer) (string, error) {
	// you can uncomment this below if you want to see exactly the commands being run
	// fmt.Println("▶️", cmd).

	command := exec.Command("sh", "-c", cmd)

	command.Dir = dir

	var outBuf bytes.Buffer

	if silent {
		command.Stdout = &outBuf
		command.Stderr = &outBuf
	} else if writer != nil {
		command.Stdout = io.MultiWriter(os.Stdout, &outBuf, writer)
		command.Stderr = io.MultiWriter(os.Stderr, &outBuf, writer)
	} else {
		command.Stdout = io.MultiWriter(os.Stdout, &outBuf)
		command.Stderr = io.MultiWriter(os.Stderr, &outBuf)
	}

	runErr := command.Run()

	outStr := outBuf.String()

	if runErr != nil {
		return outStr, errors.Wrap(runErr, "failed to Run command")
	}

	return outStr, nil
}

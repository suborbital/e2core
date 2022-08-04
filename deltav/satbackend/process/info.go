package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Info is a struct that is written to a file that describes our own process
type Info struct {
	Port int    `json:"port"`
	FQFN string `json:"fqfn"`
	PID  int    `json:"pid"`
}

// NewInfo creates an Info for the current process
func NewInfo(port int, FQFN string) *Info {
	pid := os.Getpid()

	p := &Info{
		Port: port,
		FQFN: FQFN,
		PID:  pid,
	}

	return p
}

// Find finds a process info file with the given UUID
func Find(uuid string) (*Info, error) {
	dir, err := processInfoDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to processInfoDir")
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.json", uuid))

	infoBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to ReadFile")
	}

	var info Info
	if err = json.Unmarshal(infoBytes, &info); err != nil {
		return nil, errors.Wrap(err, "failed to Unmarshal")
	}

	return &info, nil
}

// Delete deletes the process file with the given UUID if it exists
func Delete(uuid string) error {
	dir, err := processInfoDir()
	if err != nil {
		return errors.Wrap(err, "failed to processInfoDir")
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.json", uuid))

	if _, err = os.Stat(filePath); err != nil {
		// nothing to do
		return nil
	}

	if err = os.Remove(filePath); err != nil {
		return errors.Wrap(err, "failed to Remove")
	}

	return nil
}

// Write writes the Info to disk
func (p *Info) Write(uuid string) error {
	dir, err := processInfoDir()
	if err != nil {
		return errors.Wrap(err, "failed to processInfoDir")
	}

	processJSON, err := json.Marshal(p)
	if err != nil {
		return errors.Wrap(err, "failed to Marshal")
	}

	filePath := filepath.Join(dir, fmt.Sprintf("%s.json", uuid))

	if err = os.WriteFile(filePath, processJSON, 0755); err != nil {
		return errors.Wrap(err, "failed to WriteFile")
	}

	return nil
}

// processInfoDir returns the directory that Info files should be written to
func processInfoDir() (string, error) {
	config, err := os.UserConfigDir()
	if err != nil {
		return "", errors.Wrap(err, "failed to UserConfigDir")
	}

	dir := filepath.Join(config, "suborbital", "proc")

	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", errors.Wrap(err, "failed to MkdirAll")
	}

	return dir, nil
}

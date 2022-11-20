package util

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Mkdir creates a new directory to contain a runnable.
func Mkdir(cwd, name string) (string, error) {
	path := filepath.Join(cwd, name)

	if err := os.Mkdir(path, PermDirectory); err != nil {
		return "", errors.Wrap(err, "failed to Mkdir")
	}

	return path, nil
}

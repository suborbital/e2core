package util

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const CacheBaseDir = "suborbital"

// CacheDir returns the cache directory and creates it if it doesn't exist. If
// no subdirectories are passed it defaults to `suborbital/subo`.
func CacheDir(subdirectories ...string) (string, error) {
	tmpPath := os.TempDir()
	basePath, err := os.UserCacheDir()

	if err != nil {
		// fallback if $HOME is not set.
		basePath = tmpPath
	}

	base := []string{basePath, CacheBaseDir}

	if len(subdirectories) == 0 {
		base = append(base, "subo")
	}

	targetPath := filepath.Join(append(base, subdirectories...)...)

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		if err := os.MkdirAll(targetPath, PermDirectory); err != nil {
			return "", errors.Wrap(err, "failed to MkdirAll")
		}
	}

	return targetPath, nil
}

package util

import (
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
)

// ErrVersionNotPresent are errors related to checking for version numbers.
var ErrVersionNotPresent = errors.New("expected version number is not present")

// CheckFileForVersionString returns an error if the requested file does not contain the provided versionString.
func CheckFileForVersionString(filePath string, versionString string) error {
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		return errors.Wrap(err, "failed to ReadFile")
	}

	if !strings.Contains(string(file), versionString) {
		// also check if it exists without the 'v' prefix.
		noV := strings.TrimPrefix(versionString, "v")

		if !strings.Contains(string(file), noV) {
			return ErrVersionNotPresent
		}
	}

	return nil
}

package util

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
)

func getTokenPath() (string, error) {
	tokenPath, err := CacheDir("compute")
	if err != nil {
		return "", errors.Wrap(err, `failed to CacheDir("compute")`)
	}

	return filepath.Join(tokenPath, "envtoken"), nil
}

func WriteEnvironmentToken(tokenStr string) error {
	tokenPath, err := getTokenPath()
	if err != nil {
		return errors.Wrap(err, "failed to getTokenPath")
	}

	if err := ioutil.WriteFile(tokenPath, []byte(tokenStr), PermFilePrivate); err != nil {
		return errors.Wrapf(err, "failed to write %s", tokenPath)
	}

	return nil
}

func ReadEnvironmentToken() (string, error) {
	tokenPath, err := getTokenPath()
	if err != nil {
		return "", errors.Wrap(err, "failed to getTokenPath")
	}

	buf, err := ioutil.ReadFile(tokenPath)

	if err != nil {
		return "", errors.Wrapf(err, "failed to read %s", tokenPath)
	}

	return string(buf), nil
}

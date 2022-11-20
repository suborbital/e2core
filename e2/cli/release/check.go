package release

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-github/v41/github"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"

	"github.com/suborbital/e2core/e2/cli/util"
)

const lastCheckedFilename = "subo_last_checked"
const latestReleaseFilename = "subo_latest_release"

func getTimestampCache() (time.Time, error) {
	cachePath, err := util.CacheDir()
	if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to CacheDir")
	}

	cachedTimestamp := time.Time{}
	filePath := filepath.Join(cachePath, lastCheckedFilename)
	if _, err = os.Stat(filePath); os.IsNotExist(err) {
	} else if err != nil {
		return time.Time{}, errors.Wrap(err, "failed to Stat")
	} else {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return time.Time{}, errors.Wrap(err, "failed to ReadFile")
		}

		cachedTimestamp, err = time.Parse(time.RFC3339, string(data))
		if err != nil {
			errRemove := os.Remove(filePath)
			if errRemove != nil {
				return time.Time{}, errors.Wrap(err, "failed to Remove bad cached timestamp")
			}
			return time.Time{}, errors.Wrap(err, "failed to parse cached timestamp")
		}
	}
	return cachedTimestamp, nil
}

func cacheTimestamp(timestamp time.Time) error {
	cachePath, err := util.CacheDir()
	if err != nil {
		return errors.Wrap(err, "failed to CacheDir")
	}

	filePath := filepath.Join(cachePath, lastCheckedFilename)
	data := []byte(timestamp.Format(time.RFC3339))
	if err := ioutil.WriteFile(filePath, data, util.PermFile); err != nil {
		return errors.Wrap(err, "failed to WriteFile")
	}

	return nil
}

func getLatestReleaseCache() (*github.RepositoryRelease, error) {
	if cachedTimestamp, err := getTimestampCache(); err != nil {
		return nil, errors.Wrap(err, "failed to getTimestampCache")
	} else if currentTimestamp := time.Now().UTC(); cachedTimestamp.IsZero() || currentTimestamp.After(cachedTimestamp.Add(time.Hour)) {
		// check if 1 hour has passed since the last version check, and update the cached timestamp and latest release if so.
		if err := cacheTimestamp(currentTimestamp); err != nil {
			return nil, errors.Wrap(err, "failed to cacheTimestamp")
		}

		return nil, nil
	}

	cachePath, err := util.CacheDir()
	if err != nil {
		return nil, errors.Wrap(err, "failed to CacheDir")
	}

	var latestRepoRelease *github.RepositoryRelease
	filePath := filepath.Join(cachePath, latestReleaseFilename)
	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "faild to Stat")
	} else {
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to ReadFile")
		}

		var buffer bytes.Buffer
		buffer.Write(data)
		decoder := gob.NewDecoder(&buffer)
		err = decoder.Decode(&latestRepoRelease)
		if err != nil {
			errRemove := os.Remove(filePath)
			if errRemove != nil {
				return nil, errors.Wrap(err, "failed to Remove bad cached RepositoryRelease")
			}
			return nil, errors.Wrap(err, "failed to Decode cached RepositoryRelease")
		}
	}

	return latestRepoRelease, nil
}

func cacheLatestRelease(latestRepoRelease *github.RepositoryRelease) error {
	cachePath, err := util.CacheDir()
	if err != nil {
		return errors.Wrap(err, "failed to CacheDir")
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err = encoder.Encode(latestRepoRelease); err != nil {
		return errors.Wrap(err, "failed to Encode RepositoryRelease")
	} else if err := ioutil.WriteFile(filepath.Join(cachePath, latestReleaseFilename), buffer.Bytes(), util.PermFile); err != nil {
		return errors.Wrap(err, "failed to WriteFile")
	}

	return nil
}

func getLatestVersion(ctx context.Context) (*version.Version, error) {
	latestRepoRelease, err := getLatestReleaseCache()
	if err != nil {
		return nil, errors.Wrap(err, "failed to getTimestampCache")
	} else if latestRepoRelease == nil {
		latestRepoRelease, _, err = github.NewClient(nil).Repositories.GetLatestRelease(ctx, "suborbital", "e2core")
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch latest e2 release")
		} else if err = cacheLatestRelease(latestRepoRelease); err != nil {
			return nil, errors.Wrap(err, "failed to cacheLatestRelease")
		}
	}

	latestVersion, err := version.NewVersion(*latestRepoRelease.TagName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse latest e2 version")
	}

	return latestVersion, nil
}

// CheckForLatestVersion returns an error if E2CLIDotVersion does not match the latest GitHub release or if the check fails.
func CheckForLatestVersion(ctx context.Context) (string, error) {
	if latestCmdVersion, err := getLatestVersion(ctx); err != nil {
		return "", errors.Wrap(err, "failed to getLatestVersion")
	} else if cmdVersion, err := version.NewVersion(E2CLIDotVersion); err != nil {
		return "", errors.Wrap(err, "failed to parse current e2 version")
	} else if cmdVersion.LessThan(latestCmdVersion) {
		return fmt.Sprintf("An upgrade for Subo is available: %s → %s. "+
			"The method for upgrading depends on the method used for"+
			" installation (see https://github."+
			"com/suborbital/e2core for details). As always, "+
			"feel free to ping us on Discord if you run into any snags! https://chat.suborbital.dev/",
			cmdVersion, latestCmdVersion), nil
	}

	return "", nil
}

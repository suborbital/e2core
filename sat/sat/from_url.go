package sat

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func downloadFromURL(URL string) (string, error) {
	urlObj, err := url.Parse(URL)
	if err != nil {
		return "", errors.Wrap(err, "failed to url.Parse")
	}

	name := filepath.Base(urlObj.Path)

	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to NewRequest")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "failed to Do request")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to download with status code: %d", resp.StatusCode)
	}

	tmp := os.TempDir()
	dir := filepath.Join(tmp, "suborbital", "blocks")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return "", errors.Wrap(err, "failed to MkdirAll")
	}

	filename := filepath.Join(dir, name)

	file, err := os.Create(filename)
	if err != nil {
		return "", errors.Wrap(err, "failed to Open file")
	}

	defer resp.Body.Close()
	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", errors.Wrap(err, "failed to Copy file")
	}

	return filename, nil
}

func isURL(val string) bool {
	URL, err := url.Parse(val)
	if err != nil {
		return false
	}

	if URL.Host != "" && URL.Scheme == "https" && strings.HasSuffix(URL.Path, ".wasm") {
		return true
	}

	return false
}

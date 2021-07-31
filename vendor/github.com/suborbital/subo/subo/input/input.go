package input

import (
	"bufio"
	"os"

	"github.com/pkg/errors"
)

// ReadStdinString reads a string from stdin
func ReadStdinString() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	if err := scanner.Err(); err != nil {
		return "", errors.Wrap(err, "failed to scanner.Scan")
	}

	return scanner.Text(), nil
}

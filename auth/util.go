package auth

import (
	"strings"
)

func SplitIdentifier(identifier string) (string, string) {
	splitAt := strings.LastIndex(identifier, ".")
	if splitAt == -1 {
		return "", ""
	}

	return identifier[:splitAt], identifier[splitAt+1:]
}

package capabilities

import (
	"os"
	"strings"
)

func AugmentedValFromEnv(original string) string {
	val := original

	if strings.HasPrefix(original, "env(") && strings.HasSuffix(original, ")") {
		envKey := strings.TrimPrefix(original, "env(")
		envKey = strings.TrimSuffix(envKey, ")")

		val = os.Getenv(envKey)
	}

	return val
}

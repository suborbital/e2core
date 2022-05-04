package appsource

import "crypto/sha256"

// TokenHash creates a SHA-256 hash of the given string.
func TokenHash(token string) []byte {
	hasher := sha256.New()
	hasher.Write([]byte(token))

	hashBytes := hasher.Sum(nil)

	return hashBytes
}

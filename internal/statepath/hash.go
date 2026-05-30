package statepath

import (
	"crypto/sha256"
	"encoding/hex"
)

// ID returns the first 32 hex chars of sha256(input). Used for Docker-volume-
// style directory names under StateRoot.
func ID(input string) string {
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:])[:32]
}

package checksum

import (
	"crypto/sha256"
	"encoding/hex"
)

// Sum returns the hex-encoded SHA-256 digest of data.
func Sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

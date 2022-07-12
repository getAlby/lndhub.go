package security

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashPassword : Hash Password
func HashPassword(password string) string {
	bytes := sha256.Sum256([]byte(password))
	password = hex.EncodeToString(bytes[:])

	return password
}

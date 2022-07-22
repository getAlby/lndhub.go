package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const (
	PASS_SALT = "spicy salt for lndhub jLrtux4m-9FtCzNi"
)

// HashPassword : Hash Password
func HashPassword(password string) string {
	h := hmac.New(sha256.New, []byte(PASS_SALT))
	h.Write([]byte(password))
	password = hex.EncodeToString(h.Sum(nil))

	return password
}

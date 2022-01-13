package security

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword : Hash Password
func HashPassword(password *string) {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	*password = string(bytes)
}

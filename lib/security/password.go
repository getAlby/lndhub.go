package security

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword encrypts a plaintext password. Returns the ciphertext
func HashPassword(password string) string {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	password = string(bytes)

	return password
}

// VerifyPassword returns true if the ciphertext provided is a valid hash from the plaintext pass provided. False otherwise
func VerifyPassword(ciphertext, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(ciphertext), []byte(plaintext)) == nil
}

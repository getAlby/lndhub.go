package security

import (
	"crypto/sha256"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword encrypts a plaintext password. Returns the ciphertext
func HashPassword(password string) string {
	//Since bcrypt can take up to 72 bytes we hash the password first
	fixedSizePass := sha256.Sum256([]byte(password))
	bytes, _ := bcrypt.GenerateFromPassword(fixedSizePass[:], bcrypt.DefaultCost)
	password = string(bytes)

	return password
}

// VerifyPassword returns true if the ciphertext provided is a valid hash from the plaintext pass provided. False otherwise
func VerifyPassword(ciphertext, plaintext string) bool {
	//Since bcrypt can take up to 72 bytes we hash the password first
	fixedSizePass := sha256.Sum256([]byte(plaintext))
	return bcrypt.CompareHashAndPassword([]byte(ciphertext), fixedSizePass[:]) == nil
}

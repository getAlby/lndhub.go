package models

import (
	"os"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
)

// User : User Model
type User struct {
	gorm.Model
	Id           int
	Email        string
	Login        string
	Password     string
	RefreshToken string
	AccessToken  string
	UpdatedAt    int64
	CreatedAt    int64
}

var (
	jwtKey = os.Getenv("JWT_KEY")
)

// HashPassword : Hash Password
func (u *User) HashPassword() {
	bytes, _ := bcrypt.GenerateFromPassword([]byte(u.PasswordHash), bcrypt.DefaultCost)
	u.PasswordHash = string(bytes)
}

// GenerateToken : Generate Token
func (u *User) GenerateToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": u.Username,
	})

	tokenString, err := token.SignedString(jwtKey)
	return tokenString, err
}

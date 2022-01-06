package models

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
	"time"
)

// User : User Model
type User struct {
	ID           uint `gorm:"primary_key"`
	Email        string
	Login        string
	Password     string
	RefreshToken string
	AccessToken  string
	UpdatedAt    time.Time
	CreatedAt    time.Time
}

// GenerateAccessToken : Generate Access Token
func (u *User) GenerateAccessToken(c echo.Context) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id": u.ID,
		"email": u.Email,
	})

	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return err
	}
	u.AccessToken = t
	return err
}

func (u *User) GenerateRefreshToken(c echo.Context) error {
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = 1

	rt, err := refreshToken.SignedString([]byte("secret"))
	if err != nil {
		return err
	}
	u.RefreshToken = rt

	return err
}
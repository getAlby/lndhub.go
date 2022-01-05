package models

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"
)

// User : User Model
type User struct {
	gorm.Model
	Id           uint `gorm:"primary_key"`
	Email        string
	Login        string
	Password     string
	RefreshToken string
	AccessToken  string
	UpdatedAt    time.Time
	CreatedAt    time.Time
}

// GenerateToken : Generate Token
func (u *User) GenerateToken(c echo.Context) error {
	if u.Id == 0 {
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"id": u.Id,
		})

		t, err := token.SignedString([]byte("secret"))
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]string{
			u.AccessToken: t,
		})
	}

	return echo.ErrUnauthorized
}

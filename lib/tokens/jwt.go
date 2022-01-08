package tokens

import (
	"database/sql"

	"github.com/bumi/lndhub.go/database/models"
	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

type jwtCustomClaims struct {
	ID    uint   `json:"id"`
	Email string `json:"email"`
	Login string `json:"login"`

	jwt.StandardClaims
}

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(c echo.Context, u *models.User) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    u.ID,
		"email": u.Email,
		"login": u.Login,
	})

	t, err := token.SignedString([]byte("secret"))
	if err != nil {
		return err
	}
	u.AccessToken = sql.NullString{String: t}

	return err
}

func GenerateRefreshToken(c echo.Context, u *models.User) error {
	refreshToken := jwt.New(jwt.SigningMethodHS256)
	rtClaims := refreshToken.Claims.(jwt.MapClaims)
	rtClaims["sub"] = u.ID

	rt, err := refreshToken.SignedString([]byte("secret"))
	if err != nil {
		return err
	}

	u.RefreshToken = sql.NullString{String: rt}

	return err
}

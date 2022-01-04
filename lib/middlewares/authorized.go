package middlewares

import (
	"fmt"
	"net/http"
	"os"

	"github.com/dgrijalva/jwt-go"
	"github.com/labstack/echo/v4"
)

var (
	jwtKey = os.Getenv("JWT_KEY")
)

// Authoriszed : Check Auth
func Authoriszed(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie("token")
		if err != nil {
			return c.NoContent(http.StatusUnauthorized)
		}

		token, err := jwt.Parse(cookie.Value, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected Signing Method: %v", token.Header["alg"])
			}

			return jwtKey, nil
		})

		if !token.Valid || err != nil {
			return c.NoContent(http.StatusUnauthorized)
		}

		c.Set("username", token.Claims.(jwt.MapClaims)["username"])

		return next(c)
	}
}

package tokens

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bumi/lndhub.go/db/models"
	"github.com/bumi/lndhub.go/lib"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
)

func Middleware(secret []byte) echo.MiddlewareFunc {
	config := middleware.DefaultJWTConfig

	config.ContextKey = "UserJwt"
	config.SigningKey = secret
	config.SuccessHandler = func(c echo.Context) {
		token := c.Get("UserJwt").(*jwt.Token)
		claims := token.Claims.(jwt.MapClaims)
		c.Set("UserID", claims["id"])

	}

	return middleware.JWTWithConfig(config)
}

func UserMiddleware(db *bun.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.(lib.LndhubContext)
			userId := c.Get("UserID").(int64)

			var user models.User

			err := db.NewSelect().Model(&user).Where("id = ?", userId).Scan(context.TODO())
			switch {
			case errors.Is(err, sql.ErrNoRows):
				return echo.NewHTTPError(http.StatusNotFound, "user with given ID is not found")
			case err != nil:
				logrus.Errorf("database error: %v", err)
				return echo.NewHTTPError(http.StatusInternalServerError)
			}

			ctx.User = &user

			return nil
		}
	}
}

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(secret []byte, u *models.User) (string, error) {
	claims := jwt.MapClaims{
		"id":  u.ID,
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(),
	}
	if err := claims.Valid(); err != nil {
		return "", fmt.Errorf("invalid claims: %e", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return t, nil
}

// GenerateRefreshToken : Generate Refresh Token
func GenerateRefreshToken(secret []byte, u *models.User) (string, error) {
	claims := jwt.MapClaims{
		"id":         u.ID,
		"exp":        time.Now().Add(time.Hour * 24 * 30).Unix(),
		"is_refresh": true,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if err := claims.Valid(); err != nil {
		return "", fmt.Errorf("invalid claims: %e", err)
	}

	t, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return t, nil
}

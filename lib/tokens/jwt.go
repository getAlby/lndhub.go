package tokens

import (
	"context"
	"database/sql"
	"errors"
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

type jwtCustomClaims struct {
	ID        int64 `json:"id"`
	IsRefresh bool  `json:isRefresh`
	jwt.StandardClaims
}

func Middleware(secret []byte) echo.MiddlewareFunc {
	config := middleware.DefaultJWTConfig

	config.Claims = &jwtCustomClaims{}
	config.ContextKey = "UserJwt"
	config.SigningKey = secret
	config.ErrorHandler = func(err error) error {
		return echo.NewHTTPError(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    1,
			"message": "bad auth",
		})
	}
	config.SuccessHandler = func(c echo.Context) {
		token := c.Get("UserJwt").(*jwt.Token)
		claims := token.Claims.(*jwtCustomClaims)
		c.Set("UserID", claims.ID)

	}

	return middleware.JWTWithConfig(config)
}

func UserMiddleware(db *bun.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.(*lib.LndhubContext)
			userId := c.Get("UserID")

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

			return next(c)
		}
	}
}

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(secret []byte, u *models.User) (string, error) {
	claims := &jwtCustomClaims{
		u.ID,
		false,
		jwt.StandardClaims{
			// one week expiration
			ExpiresAt: time.Now().Add(time.Hour * 24 * 7).Unix(),
		},
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
	claims := &jwtCustomClaims{
		u.ID,
		true,
		jwt.StandardClaims{
			// one week expiration
			ExpiresAt: time.Now().Add(time.Hour * 24 * 7).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return t, nil
}

package tokens

import (
	"errors"
	"net/http"
	"time"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type jwtCustomClaims struct {
	ID        int64 `json:"id"`
	IsRefresh bool  `json:"isRefresh"`
	jwt.StandardClaims
}

func Middleware(secret []byte) echo.MiddlewareFunc {
	config := middleware.DefaultJWTConfig

	config.Claims = &jwtCustomClaims{}
	config.ContextKey = "UserJwt"
	config.SigningKey = secret
	config.ErrorHandlerWithContext = func(err error, c echo.Context) error {
		c.Logger().Error(err)
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

// GenerateAccessToken : Generate Access Token
func GenerateAccessToken(secret []byte, expiryInSeconds int, u *models.User) (string, error) {
	claims := &jwtCustomClaims{
		ID:        u.ID,
		IsRefresh: false,
		StandardClaims: jwt.StandardClaims{
			// one week expiration
			ExpiresAt: time.Now().Add(time.Second * time.Duration(expiryInSeconds)).Unix(),
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
func GenerateRefreshToken(secret []byte, expiryInSeconds int, u *models.User) (string, error) {
	claims := &jwtCustomClaims{
		ID:        u.ID,
		IsRefresh: true,
		StandardClaims: jwt.StandardClaims{
			// one week expiration
			ExpiresAt: time.Now().Add(time.Second * time.Duration(expiryInSeconds)).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	t, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return t, nil
}
func ParseToken(secret []byte, token string) (int64, error) {
	userIdClaim := "id"
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return -1, err
	}

	if !parsedToken.Valid {
		return -1, errors.New("Token is invalid")
	}

	var userId interface{}
	for k, v := range claims {
		if k == userIdClaim {
			userId = v.(float64)
		}
	}

	if userId == nil {
		return -1, errors.New("User id claim not found")
	}

	return int64(userId.(float64)), nil
}

func GetUserIdFromToken(secret []byte, token string) (int64, error) {
	userIdClaim := "id"
	isRefreshClaim := "isRefresh"
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})

	if err != nil {
		return -1, err
	}

	if !parsedToken.Valid {
		return -1, errors.New("Token is invalid")
	}

	var userId interface{}
	for k, v := range claims {
		if k == isRefreshClaim && v.(bool) == false {
			return -1, errors.New("This is not a refresh token")
		}
		if k == userIdClaim {
			userId = v.(float64)
		}
	}

	if userId == nil {
		return -1, errors.New("User id claim not found")
	}

	return int64(userId.(float64)), nil
}

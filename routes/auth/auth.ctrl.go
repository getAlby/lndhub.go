package auth

import (
	"net/http"

	"github.com/bumi/lndhub.go/database/models"
	"github.com/bumi/lndhub.go/lib/tokens"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// AuthRouter : AuthRouter struct
type AuthRouter struct{}

// Auth : Auth Router
func (AuthRouter) Auth(c echo.Context) error {
	type RequestBody struct {
		Login        string `json:"login"`
		Password     string `json:"password"`
		RefreshToken string `json:"refresh_token"`
	}

	var body RequestBody

	if err := c.Bind(&body); err != nil {
		return err
	}

	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "Bad arguments",
		})
	}

	if (body.Login == "" || body.Password == "") && body.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "login and password or refresh token is required",
		})
	}

	db, _ := c.Get("db").(*gorm.DB)

	var user models.User

	if body.Login != "" || body.Password != "" {
		if err := db.Where("login = ? AND password = ?", body.Login, body.Password).First(&user).Error; err != nil {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":   true,
				"code":    1,
				"message": "bad auth",
			})
		}
	} else if body.RefreshToken != "" {
		if err := db.Where("refresh_token = ?", body.RefreshToken).First(&user).Error; err != nil {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":   true,
				"code":    1,
				"message": "bad auth",
			})
		}
	}

	err := tokens.GenerateAccessToken(c, &user)
	if err != nil {
		return err
	}
	err = tokens.GenerateRefreshToken(c, &user)
	if err != nil {
		return err
	}

	if err := db.Save(&user).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error":   true,
			"code":    6,
			"message": "Something went wrong. Please try again later",
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"refresh_token": user.RefreshToken.String,
		"access_token":  user.AccessToken.String,
	})
}

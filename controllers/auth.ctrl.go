package controllers

import (
	"net/http"

	"github.com/bumi/lndhub.go/db/models"
	"github.com/bumi/lndhub.go/lib/tokens"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/labstack/echo/v4"
)

// AuthController : AuthController struct
type AuthController struct{}

// Auth : Auth Controller
func (AuthController) Auth(c echo.Context) error {
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
		if err := db.Where("login = ?", body.Login).First(&user).Error; err != nil {
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":   true,
				"code":    1,
				"message": "bad auth",
			})
		}
		if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
			return c.JSON(http.StatusNotFound, echo.Map{
				"message": "invalid username or password",
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

	accessToken, err := tokens.GenerateAccessToken(&user)
	if err != nil {
		return err
	}
	refreshToken, err := tokens.GenerateRefreshToken(&user)
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
		"refresh_token": refreshToken,
		"access_token":  accessToken,
	})
}

package auth

import (
	"net/http"

	"github.com/bumi/lndhub.go/database/models"
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
		return err
	}

	if (body.Login == "" && body.Password == "") && body.RefreshToken == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "login and password or refresh token is required",
		})
	}

	db, _ := c.Get("db").(*gorm.DB)

	var user models.User

	if body.Login != "" && body.Password != "" {
		if err := db.Where("login = ? AND password = ?", body.Login, body.Password).First(&user).Error; err != nil {
			return c.JSON(http.StatusNotFound, echo.Map{
				"message": "user not found",
			})
		}
	} else if body.RefreshToken != "" {
		if err := db.Where("refresh_token = ?", body.RefreshToken).First(&user).Error; err != nil {
			return c.JSON(http.StatusNotFound, echo.Map{
				"message": "user not found",
			})
		}
	}

	err := user.GenerateAccessToken(c)
	if err != nil {
		return err
	}
	err = user.GenerateRefreshToken(c)
	if err != nil {
		return err
	}

	if err := db.Model(&user).Where("id = ?", user.ID).Update("access_token", user.AccessToken).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "server error, try again",
		})
	}
	if err := db.Model(&user).Where("id = ?", user.ID).Update("refresh_token", user.RefreshToken).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "server error, try again",
		})
	}
	//var cookie http.Cookie
	//
	//cookie.Name = "token"
	//cookie.Value = user.AccessToken
	//cookie.Expires = time.Now().Add(7 * 24 * time.Hour)
	//
	//c.SetCookie(&cookie)

	return c.JSON(http.StatusOK, echo.Map{
		"user": user,
	})
}

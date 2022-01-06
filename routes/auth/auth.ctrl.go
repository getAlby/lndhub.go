package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/bumi/lndhub.go/database/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// Register : Register Router
func (AuthRouter) Auth(c echo.Context) error {
	type RequestBody struct {
		Id       int    `json:"id" validate:"required"`
		Email    string `json:"email" validate:"required"`
		Password string `json:"password" validate:"required"`
	}

	var body RequestBody
	var err error
	body.Id, err = strconv.Atoi(c.FormValue("id"))
	if err != nil {
		return err
	}
	body.Email = c.FormValue("email")
	body.Password = c.FormValue("password")

	if err := c.Validate(&body); err != nil {
		return err
	}

	db, _ := c.Get("db").(*gorm.DB)

	var user models.User

	if err := db.Where("id = ?", body.Id).First(&user).Error; err != nil {
		return c.NoContent(http.StatusConflict)
	}

	err = user.GenerateAccessToken(c, &user)
	if err != nil {
		return err
	}

	var cookie http.Cookie

	cookie.Name = "token"
	cookie.Value = user.AccessToken
	cookie.Expires = time.Now().Add(7 * 24 * time.Hour)

	c.SetCookie(&cookie)

	return c.JSON(http.StatusOK, echo.Map{
		"token": user.AccessToken,
		"user":  user,
	})
}

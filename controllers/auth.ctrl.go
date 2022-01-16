package controllers

import (
	"context"
	"net/http"

	"github.com/bumi/lndhub.go/db/models"
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lib/tokens"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// AuthController : AuthController struct
type AuthController struct{}

// Auth : Auth Controller
func (AuthController) Auth(c echo.Context) error {
	ctx := c.(*lib.LndhubContext)
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

	db := ctx.DB
	var user models.User

	switch {
	case body.Login != "" || body.Password != "":
		{
			if err := db.NewSelect().Model(&user).Where("login = ?", body.Login).Scan(context.TODO()); err != nil {
				return c.JSON(http.StatusNotFound, echo.Map{
					"error":   true,
					"code":    1,
					"message": "bad auth",
				})
			}
			if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password)) != nil {
				return c.JSON(http.StatusNotFound, echo.Map{
					"error":   true,
					"code":    1,
					"message": "bad auth",
				})
			}
		}
	case body.RefreshToken != "":
		{
			// TODO: currently not supported
			// I'd love to remove this from the auth handler, as the refresh token
			// is usually a part of the JWT middleware: https://webdevstation.com/posts/user-authentication-with-go-using-jwt-token/
			// if the current client depends on that - we can incorporate the refresh JWT code into here
			return c.JSON(http.StatusNotFound, echo.Map{
				"error":   true,
				"code":    1,
				"message": "bad auth",
			})
		}
	default:
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "login and password or refresh token is required",
		})
	}

	accessToken, err := tokens.GenerateAccessToken(&user)
	if err != nil {
		return err
	}

	refreshToken, err := tokens.GenerateRefreshToken(&user)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"refresh_token": refreshToken,
		"access_token":  accessToken,
	})
}

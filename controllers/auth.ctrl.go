package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// AuthController : AuthController struct
type AuthController struct {
	JWTSecret []byte
	JWTExpiry int
	svc       *lib.LndhubService
}

func NewAuthController(svc *lib.LndhubService) *AuthController {
	return &AuthController{
		svc:       svc,
		JWTSecret: svc.Config.JWTSecret,
		JWTExpiry: svc.Config.JWTExpiry,
	}
}

// Auth : Auth Controller
func (controller *AuthController) Auth(c echo.Context) error {
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

	accessToken, refreshToken, err := controller.svc.GenerateToken(body.Login, body.Password, body.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, echo.Map{
		"refresh_token": refreshToken,
		"access_token":  accessToken,
	})
}

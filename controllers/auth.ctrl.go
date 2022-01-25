package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// AuthController : AuthController struct
type AuthController struct {
	svc *service.LndhubService
}

func NewAuthController(svc *service.LndhubService) *AuthController {
	return &AuthController{
		svc: svc,
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
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	accessToken, refreshToken, err := controller.svc.GenerateToken(body.Login, body.Password, body.RefreshToken)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, echo.Map{
		"refresh_token": refreshToken,
		"access_token":  accessToken,
	})
}

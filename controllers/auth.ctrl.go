package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)
// TODO adjust AuthController to verify that the signature on
//		the event being handled is:
// 		* valid for the pubkey on the same event
//		* matches a pubkey of one of our users

// AuthController : AuthController struct
type AuthController struct {
	svc *service.LndhubService
}

func NewAuthController(svc *service.LndhubService) *AuthController {
	return &AuthController{
		svc: svc,
	}
}

type AuthRequestBody struct {
	Pubkey       string `json:"pubkey"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token"`
}
type AuthResponseBody struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// Auth godoc
// @Summary      Authenticate
// @Description  Exchanges a login + password for a token
// @Accept       json
// @Produce      json
// @Tags         Account
// @Param        AuthRequestBody  body      AuthRequestBody  false  "Login and password"
// @Success      200              {object}  AuthResponseBody
// @Failure      400              {object}  responses.ErrorResponse
// @Failure      500              {object}  responses.ErrorResponse
// @Router       /auth [post]
func (controller *AuthController) Auth(c echo.Context) error {

	var body AuthRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load auth user request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	if err := c.Validate(&body); err != nil {
		c.Logger().Errorj(
			log.JSON{
				"error":   err,
				"message": "invalid request body",
			},
		)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if body.Pubkey == "" || body.Password == "" {
		// To support Swagger we also look in the Form data
		params, err := c.FormParams()
		if err != nil {
			c.Logger().Errorj(
				log.JSON{
					"message": "invalid form parameters",
					"error":   err,
				},
			)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
		pubkey := params.Get("pubkey")
		password := params.Get("password")
		if pubkey != "" && password != "" {
			body.Pubkey = pubkey
			body.Password = password
		}
	}

	accessToken, refreshToken, err := controller.svc.GenerateToken(c.Request().Context(), body.Pubkey, body.Password, body.RefreshToken)
	if err != nil {
		if err.Error() == responses.AccountDeactivatedError.Message {
			c.Logger().Errorj(
				log.JSON{
					"message":    "account deactivated",
					"user_pubkey": body.Pubkey,
				},
			)
			return c.JSON(http.StatusUnauthorized, responses.AccountDeactivatedError)
		}
		c.Logger().Errorj(
			log.JSON{
				"message":    "authentication error",
				"user_pubkey": body.Pubkey,
				"error":      err,
			},
		)
		return c.JSON(http.StatusUnauthorized, responses.BadAuthError)
	}

	return c.JSON(http.StatusOK, &AuthResponseBody{
		RefreshToken: refreshToken,
		AccessToken:  accessToken,
	})
}

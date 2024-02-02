package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// CreateUserController : Create user controller struct
type CreateUserController struct {
	svc *service.LndhubService
}

func NewCreateUserController(svc *service.LndhubService) *CreateUserController {
	return &CreateUserController{svc: svc}
}

type CreateUserResponseBody struct {
	Pubkey    string `json:"pubkey"`
}
type CreateUserRequestBody struct {
	Pubkey       string `json:"pubkey"`
}

func (controller *CreateUserController) CreateUser(c echo.Context) error {

	var body CreateUserRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load create user request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	user, err := controller.svc.CreateUser(c.Request().Context(), body.Pubkey)
	if err != nil {
		c.Logger().Errorf("Failed to create user: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	var ResponseBody CreateUserResponseBody
	ResponseBody.Pubkey = user.Pubkey

	return c.JSON(http.StatusOK, &ResponseBody)
}

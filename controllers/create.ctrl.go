package controllers

import (
	"net/http"

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

// CreateUser : Create user Controller
func (controller *CreateUserController) CreateUser(c echo.Context) error {
	// optional parameters that we currently do not use
	type RequestBody struct {
		PartnerID   string `json:"partnerid"`
		AccountType string `json:"accounttype"`
	}
	var body RequestBody

	if err := c.Bind(&body); err != nil {
		return err
	}
	user, err := controller.svc.Create()
	//todo json response
	if err != nil {
		return err
	}

	var ResponseBody struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	ResponseBody.Login = user.Login
	ResponseBody.Password = user.Password

	return c.JSON(http.StatusOK, &ResponseBody)
}

package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// CreateUserController : Create user controller struct
type CreateUserController struct {
	svc    *service.LndhubService
	plugin func(CreateUserResponseBody, *service.LndhubService) (CreateUserResponseBody, error)
}

func NewCreateUserController(svc *service.LndhubService) *CreateUserController {
	result := &CreateUserController{svc: svc}
	//check for plugin
	if plug, ok := svc.MiddlewarePlugins["create"]; ok {
		mwPlugin := plug.Interface().(func(in CreateUserResponseBody, svc *service.LndhubService) (CreateUserResponseBody, error))
		result.plugin = mwPlugin
	}

	return result
}

type CreateUserResponseBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
type CreateUserRequestBody struct {
	Login       string `json:"login"`
	Password    string `json:"password"`
	PartnerID   string `json:"partnerid"`
	AccountType string `json:"accounttype"`
}

// CreateUser : Create user Controller
func (controller *CreateUserController) CreateUser(c echo.Context) error {

	var body CreateUserRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load create user request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	user, err := controller.svc.CreateUser(c.Request().Context(), body.Login, body.Password)
	if err != nil {
		c.Logger().Errorf("Failed to create user: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	var ResponseBody CreateUserResponseBody
	ResponseBody.Login = user.Login
	ResponseBody.Password = user.Password
	if controller.plugin != nil {
		ResponseBody, err = controller.plugin(ResponseBody, controller.svc)
		if err != nil {
			return err
		}
	}
	return c.JSON(http.StatusOK, &ResponseBody)
}

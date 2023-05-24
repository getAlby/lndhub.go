package v2controllers

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
	Login    string `json:"login"`
	Password string `json:"password"`
	ID       int64  `json:"id"`
}
type CreateUserRequestBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// CreateUser godoc
// @Summary      Create an account
// @Description  Create a new account with a login and password.
// @Accept       json
// @Produce      json
// @Tags         Account
// @Param        account  body      CreateUserRequestBody  false  "Create User"
// @Success      200      {object}  CreateUserResponseBody
// @Failure      400      {object}  responses.ErrorResponse
// @Failure      500      {object}  responses.ErrorResponse
// @Router       /v2/users [post]
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
	ResponseBody.ID = user.ID

	return c.JSON(http.StatusOK, &ResponseBody)
}

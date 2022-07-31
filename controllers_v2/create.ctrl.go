package v2controllers

import (
	"net/http"
	"strings"

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
	Nickname string `json:"nickname"`
}
type CreateUserRequestBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

// CreateUser godoc
// @Summary      Create an account
// @Description  Create a new account with a login and password. If login is an libp2p CID then the password must be the signature("log in into mintter lndhub: <accountID>)") and the pubkey must be present in the auth header.
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

	user, err := controller.svc.CreateUser(c.Request().Context(), body.Login, body.Password, body.Nickname)
	if err != nil {
		c.Logger().Errorf("Failed to create user: %v", err)
		if strings.Contains(err.Error(), responses.LoginTakenError.Message) ||
			(strings.Contains(err.Error(), "duplicate") && strings.Contains(err.Error(), "login")) {
			return c.JSON(http.StatusBadRequest, responses.LoginTakenError)
		} else if strings.Contains(err.Error(), responses.NicknameTakenError.Message) ||
			(strings.Contains(err.Error(), "duplicate") && strings.Contains(err.Error(), "nickname")) {
			return c.JSON(http.StatusBadRequest, responses.NicknameTakenError)
		} else if strings.Contains(err.Error(), responses.NicknameFormatError.Message) {
			return c.JSON(http.StatusBadRequest, responses.NicknameFormatError)
		} else {
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}

	}

	var ResponseBody CreateUserResponseBody
	ResponseBody.Login = user.Login
	ResponseBody.Password = user.Password
	ResponseBody.Nickname = user.Nickname

	return c.JSON(http.StatusOK, &ResponseBody)
}

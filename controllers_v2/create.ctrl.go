package v2controllers

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p-core/peer"
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
// @Description  Create a new account with a login and password login must be accountID and password signature("log in into mintter lndhub: <accountID>)")
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
	if body.Login != "" && body.Password != "" {
		c.Logger().Infof("Login %v, Prefix %v, Password %v", body.Login, controller.svc.Config.SignedMessagePrefix, body.Password)
		if err := checkSignature(body.Login, controller.svc.Config.SignedMessagePrefix, body.Password); err != nil {
			c.Logger().Errorf("Wrong user password combination: %v", err)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
	}
	user, err := controller.svc.CreateUser(c.Request().Context(), body.Login, body.Password, body.Nickname)
	if err != nil {
		c.Logger().Errorf("Failed to create user: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	var ResponseBody CreateUserResponseBody
	ResponseBody.Login = user.Login
	ResponseBody.Password = user.Password
	ResponseBody.Nickname = user.Nickname

	return c.JSON(http.StatusOK, &ResponseBody)
}

// checkSignature verifies that the signature provided is a valid signature
// for the public key pubkey, being the message = <fixedprefix><pubkey>
func checkSignature(accountID, prefix, signature string) error {
	id, err := peer.IDFromString(accountID)
	if err != nil {
		return err
	}
	pubKey, err := id.ExtractPublicKey()
	if err != nil {
		return err
	}
	signature_bin, err := hex.DecodeString(signature)
	if err != nil {
		return err
	}
	ok, err := pubKey.Verify([]byte(prefix+accountID), signature_bin)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("signature does not match public key")
	}
	return nil
}

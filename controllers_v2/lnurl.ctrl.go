package v2controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

const (
	MIN_SENDABLE_SATS = 1
	MAX_SENDABLE_SATS = 10000000
)

// LnurlController : Add lnurl controller struct
type LnurlController struct {
	svc *service.LndhubService
}

func NewLnurlController(svc *service.LndhubService) *LnurlController {
	return &LnurlController{svc: svc}
}

type LnurlpResponseBody struct {
	Callback    string `json:"callback"`
	MaxSendable uint64 `json:"maxSendable"`
	MinSendable uint64 `json:"minSendable"`
	Metadata    string `json:"metadata"`
	Tag         string `json:"tag"`
}

func (controller *LnurlController) Lnurlp(c echo.Context) error {
	// The user param could be userID (login) or a nickname (lnaddress)
	user, err := controller.svc.FindUserByLoginOrNickname(c.Request().Context(), c.Param("user"))
	if err != nil {
		c.Logger().Errorf("Failed to find user by login&username: user %v error %v", c.Param("user"), err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	responseBody := &LnurlpResponseBody{}
	responseBody.MinSendable = MIN_SENDABLE_SATS
	responseBody.MaxSendable = MAX_SENDABLE_SATS
	for _, s := range c.ParamNames() {
		if strings.ToLower(s) == "amt" {
			amt, err := strconv.ParseInt(c.QueryParam(s), 10, 64)
			if err != nil {
				c.Logger().Errorf("Could not convert %v to uint64. %v", c.QueryParam(s), err)
				return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
			}
			responseBody.MinSendable = uint64(amt)
			responseBody.MaxSendable = responseBody.MinSendable
			break
		}
	}

	responseBody.Callback = user.Login
	responseBody.Metadata = "[[\"text/identifier\", \"" + user.Nickname + "@mintter.com\"], [\"text/plain\", \"Sats for " + user.Nickname + "\"]]"

	responseBody.Tag = "payRequest"
	return c.JSON(http.StatusOK, &responseBody)
}

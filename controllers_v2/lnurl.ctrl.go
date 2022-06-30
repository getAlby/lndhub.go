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
	PREFIX_MSG        = "Sats for "
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

// Lnurlp godoc
// @Summary      Responds to a LNURL payRequest
// @Description  Server side (LN SERVICE) of the LUD-06 lnurl spec
// @Accept       json
// @Produce      json
// @Param        user path string true "User login or nickname"
// @Tags         Lnurl
// @Success      200  {object}  LnurlpResponseBody
// @Failure      400  {object}  responses.LnurlErrorResponse
// @Failure      500  {object}  responses.LnurlErrorResponse
// @Router       /v2/lnurlp/{user} [get]
// @Security     OAuth2Password
func (controller *LnurlController) Lnurlp(c echo.Context) error {
	// The user param could be userID (login) or a nickname (lnaddress)
	user, err := controller.svc.FindUserByLoginOrNickname(c.Request().Context(), c.Param("user"))
	if err != nil {
		c.Logger().Errorf("Failed to find user by login or nickname: user %v error %v", c.Param("user"), err)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}

	responseBody := &LnurlpResponseBody{}
	responseBody.MinSendable = MIN_SENDABLE_SATS
	responseBody.MaxSendable = MAX_SENDABLE_SATS
	for _, s := range c.ParamNames() {
		if strings.ToLower(s) == "amt" {
			amt, err := strconv.ParseInt(c.QueryParam(s), 10, 64)
			if err != nil {
				c.Logger().Errorf("Could not convert %v to uint64. %v", c.QueryParam(s), err)
				return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
			}
			responseBody.MinSendable = uint64(amt)
			responseBody.MaxSendable = responseBody.MinSendable
			break
		}
	}
	responseBody.Callback = "https://" + controller.svc.Config.LnurlAPIPrefix + "." + controller.svc.Config.LnurlDomain + "/v2/invoice/" + c.Param("user")
	responseBody.Metadata = lnurlDescriptionHash(user.Nickname, controller.svc.Config.LnurlDomain)

	responseBody.Tag = "payRequest"
	return c.JSON(http.StatusOK, &responseBody)
}

func lnurlDescriptionHash(nickname, domain string) string {
	return "[[\"text/identifier\", \"" + nickname + "@" + domain + "\"], [\"text/plain\", \"" + PREFIX_MSG + nickname + "\"]]"
}

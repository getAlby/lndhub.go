package v2controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

const (
	MIN_RECEIVABLE_MSATS = 1000
	SERVICE_CUT_BPS      = 125 // 125 basis points
	PREFIX_SINGLE_MSG    = "Sats for "
	PREFIX_TWO_MSG       = "Sat to be split between "
	PREFIX_MULTIPLE_MSG  = "Sat to be split among: "
	LNURLP_COMMENT_SIZE  = 127
	LNURLP_TAG           = "payRequest"
)

// LnurlController : Add lnurl controller struct
type LnurlController struct {
	svc *service.LndhubService
}

func NewLnurlController(svc *service.LndhubService) *LnurlController {
	return &LnurlController{svc: svc}
}

type LnurlpResponseBody struct {
	Callback       string `json:"callback"`
	MaxSendable    uint64 `json:"maxSendable"`
	MinSendable    uint64 `json:"minSendable"`
	Metadata       string `json:"metadata"`
	CommentAllowed uint   `json:"commentAllowed"`
	Tag            string `json:"tag"`
}

// Lud6Invoice godoc
// @Summary      Generate an invoice without credentials
// @Description  Ask a user to generate an invoice
// @Accept       json
// @Produce      json
// @Param        user_login path string true "User login or nickname"
// @Param        amount path string true "amount in millisatoshis at the invoice"
// @Tags         Lud6Invoice
// @Success      200  {object}  Lud6InvoiceResponseBody
// @Failure      400  {object}  responses.LnurlErrorResponse
// @Failure      500  {object}  responses.LnurlErrorResponse
// @Router       /v2/invoice [get]
// @Security     OAuth2Password
func (controller *InvoiceController) Lud6Invoice(c echo.Context) error {
	// The user param could be userID (login) or a nickname (lnaddress)

	if !c.QueryParams().Has("user") {
		c.Logger().Errorf("user mandatory param in query URL")
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	paymentMeta := PaymentMetadata{Authors: map[string]float64{}}
	cumulativeAuthorship := 0.0
	var err error
	users := []*models.User{}
	houseUser, err := controller.svc.FindUserByLogin(c.Request().Context(), controller.svc.Config.HouseUser)
	if err != nil {
		c.Logger().Errorf("Failed to find house user to collect and distribute payments on behalf of authors: %v", err)
		return c.JSON(http.StatusInternalServerError, responses.GeneralServerError)
	}
	captable := service.Captable{LeadingUserID: houseUser.ID, SecondaryUsers: map[int64]float64{}}
	c.Request().ParseForm()
	params := c.Request().Form
	fmt.Println(params)
	for _, slice := range c.QueryParams()["user"] {
		authorSlice := strings.Split(slice, ",")
		user, err := controller.svc.FindUserByLoginOrNickname(c.Request().Context(), authorSlice[0])
		if err != nil {
			c.Logger().Errorf("Failed to find user by login or nickname: user %v error %v", authorSlice[0], err)
			return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
		}
		users = append(users, user)
		if len(authorSlice) > 2 {
			c.Logger().Debugf("user param must be in the format <id>,<floatauthorship> or <id>, got %s", slice)
			return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
		}
		authorship := 1.0 - float64(SERVICE_CUT_BPS)/10000
		if len(authorSlice) == 2 {
			if authorship, err = strconv.ParseFloat(authorSlice[1], 64); err != nil {
				c.Logger().Debugf("Could not parse authorship from user: %v", c.QueryParams()["user"])
				return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
			}
		}

		cumulativeAuthorship += authorship
		paymentMeta.Authors[authorSlice[0]] = authorship
		already, ok := captable.SecondaryUsers[user.ID]
		if ok {
			captable.SecondaryUsers[user.ID] = already + authorship
		} else {
			captable.SecondaryUsers[user.ID] = authorship
		}
	}
	if cumulativeAuthorship > 1 {
		c.Logger().Debugf("Slices added up more than 1: %v", cumulativeAuthorship)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}

	records, err := json.Marshal(paymentMeta)
	if err != nil {
		c.Logger().Debugf("Could not parse to json: %v", paymentMeta)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	var amt_msat int64 = -1
	if c.QueryParams().Has("amount") {
		amt_msat, err = strconv.ParseInt(c.QueryParam("amount"), 10, 64)
		if err != nil {
			c.Logger().Errorf("Could not convert %v to uint64. %v", c.QueryParam("amount"), err)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
	}

	if amt_msat/1000 <= 0 {
		c.Logger().Errorf("Amount provided [%v] lower than minimum of 1000 msats", amt_msat/1000)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	if controller.svc.Config.MaxReceiveAmount > 0 && amt_msat/1000 > controller.svc.Config.MaxReceiveAmount {
		c.Logger().Errorf("Provided amount [%v] exceeded max receive limit [%v]", amt_msat/1000, controller.svc.Config.MaxReceiveAmount)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}

	if c.QueryParams().Has("source") {
		paymentMeta.Source = c.QueryParam("source")
	}

	var descriptionhash_string string = ""
	var memo string = ""
	if c.QueryParams().Has("memo") {
		memo = c.QueryParam("memo")
	} else {
		descriptionHash := lnurlDescriptionHash(users, controller.svc.Config.LnurlDomain)
		descriptionhash_hex := sha256.Sum256([]byte(descriptionHash))
		descriptionhash_string = hex.EncodeToString(descriptionhash_hex[:])
	}

	c.Logger().Infof("Adding invoice: value:%v description_hash:%s memo:%s", amt_msat/1000, descriptionhash_string, memo)

	invoice, err := controller.svc.AddIncomingInvoice(c.Request().Context(), captable.LeadingUserID, amt_msat/1000, memo, descriptionhash_string, records...)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: %v", err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	responseBody := Lud6InvoiceResponseBody{}
	responseBody.Payreq = invoice.PaymentRequest
	captable.Invoice = invoice
	if err := controller.svc.SplitIncomingPayment(c.Request().Context(), captable); err != nil {
		c.Logger().Errorf("Error splitting invoice: %v", err)
		if !errors.Is(err, responses.LeadAuthorIncludedError) {
			sentry.CaptureException(err)
			return c.JSON(http.StatusInternalServerError, responses.GeneralServerError)
		}
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	return c.JSON(http.StatusOK, &responseBody)

}

// Lnurlp godoc
// @Summary      Responds to a LNURL payRequest
// @Description  Server side (LN SERVICE) of the LUD-06 lnurl spec
// @Accept       json
// @Produce      json
// @Param        user path string true "User login or nickname"
// @Param        amt path string true "amount in satoshis to request invoice"
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
	responseBody.MinSendable = MIN_RECEIVABLE_MSATS
	responseBody.MaxSendable = uint64(controller.svc.Config.MaxReceiveAmount * 1000)
	responseBody.Callback = "https://" + controller.svc.Config.LnurlDomain + "/v2/invoice?user=" + c.Param("user")
	if !c.QueryParams().Has("amt") {
		c.Logger().Debug("Amount not provided")
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	amt, err := strconv.ParseInt(c.QueryParam("amt"), 10, 64)
	if err != nil {
		c.Logger().Errorf("Could not convert %v to uint64. %v", c.QueryParam(c.QueryParam("amt")), err)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	if amt > controller.svc.Config.MaxReceiveAmount || amt < MIN_RECEIVABLE_MSATS {
		c.Logger().Errorf("amt provided (%d) not in range [%d-%d] msats. %v", amt, MIN_RECEIVABLE_MSATS, controller.svc.Config.MaxReceiveAmount)
		return c.JSON(http.StatusBadRequest, responses.LnurlpBadArgumentsError)
	}
	responseBody.MinSendable = uint64(amt * 1000)
	responseBody.MaxSendable = uint64(amt * 1000)
	responseBody.Callback = responseBody.Callback + "&amount=" + strconv.FormatInt(amt*1000, 10)

	responseBody.Metadata = lnurlDescriptionHash([]*models.User{user}, controller.svc.Config.LnurlDomain)
	responseBody.CommentAllowed = LNURLP_COMMENT_SIZE
	responseBody.Tag = LNURLP_TAG
	return c.JSON(http.StatusOK, &responseBody)
}

func lnurlDescriptionHash(users []*models.User, domain string) string {
	switch len(users) {
	case 0:
		return ""
	case 1:
		return "[[\"text/email\", \"" + users[0].Nickname + "@" + domain + "\"], [\"text/plain\", \"" + PREFIX_SINGLE_MSG + users[0].Nickname + "\"]]"
	case 2:
		return "[[\"text/plain\", \"" + PREFIX_TWO_MSG + users[0].Nickname + " and " + users[1].Nickname + "\"]]"
	default:
		authors := []string{}
		for _, a := range users {
			authors = append(authors, a.Nickname)
		}
		return "[[\"text/plain\", \"" + PREFIX_MULTIPLE_MSG + strings.Join(authors[:], ", ") + "\"]]"
	}
}

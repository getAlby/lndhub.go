package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

// AddInvoiceController : Add invoice controller struct
type AddInvoiceController struct {
	svc *service.LndhubService
}

func NewAddInvoiceController(svc *service.LndhubService) *AddInvoiceController {
	return &AddInvoiceController{svc: svc}
}

type AddInvoiceRequestBody struct {
	Amount          interface{} `json:"amt"` // amount in Satoshi
	Memo            string      `json:"memo"`
	DescriptionHash string      `json:"description_hash" validate:"omitempty,hexadecimal,len=64"`
}

type AddInvoiceResponseBody struct {
	RHash          string `json:"r_hash"`
	PaymentRequest string `json:"payment_request"`
	PayReq         string `json:"pay_req"`
}

func (controller *AddInvoiceController) AddInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	return AddInvoice(c, controller.svc, userID)
}

func AddInvoice(c echo.Context, svc *service.LndhubService, userID int64) error {
	var body AddInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	amount, err := svc.ParseInt(body.Amount)
	if err != nil || amount < 0 {
		c.Logger().Errorf("Invalid amount %v for user_id:%v error: %v", amount, userID, err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if svc.Config.MaxReceiveAmount > 0 {
		if amount > svc.Config.MaxReceiveAmount {
			c.Logger().Errorf("Max receive amount exceeded for user_id:%v (amount:%v)", userID, amount)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
	}

	if svc.Config.MaxAccountBalance > 0 {
		currentBalance, err := svc.CurrentUserBalance(c.Request().Context(), userID)
		if err != nil {
			c.Logger().Errorf("Error fetching balance for user_id:%v error: %v", userID, err)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
		if currentBalance+amount > svc.Config.MaxAccountBalance {
			c.Logger().Errorf("Max account balance exceeded for user_id:%v (balance:%v + amount:%v)", userID, currentBalance, amount)
			return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
		}
	}

	c.Logger().Infof("Adding invoice: user_id:%v memo:%s value:%v description_hash:%s", userID, body.Memo, amount, body.DescriptionHash)

	invoice, err := svc.AddIncomingInvoice(c.Request().Context(), userID, amount, body.Memo, body.DescriptionHash)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: user_id:%v error: %v", userID, err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	responseBody := AddInvoiceResponseBody{}
	responseBody.RHash = invoice.RHash
	responseBody.PaymentRequest = invoice.PaymentRequest
	responseBody.PayReq = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)
}

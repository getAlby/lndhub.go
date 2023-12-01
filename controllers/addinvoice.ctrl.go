package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
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
		c.Logger().Errorj(
			log.JSON{
				"error":          err,
				"message":        "invalid amount",
				"lndhub_user_id": userID,
				"amount":         amount,
			},
		)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	c.Logger().Infof("Adding invoice: user_id:%v memo:%s value:%v description_hash:%s", userID, body.Memo, amount, body.DescriptionHash)

	invoice, errResp := svc.AddIncomingInvoice(c.Request().Context(), userID, amount, body.Memo, body.DescriptionHash)
	if errResp != nil {
		return c.JSON(errResp.HttpStatusCode, errResp)
	}
	responseBody := AddInvoiceResponseBody{}
	responseBody.RHash = invoice.RHash
	responseBody.PaymentRequest = invoice.PaymentRequest
	responseBody.PayReq = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)
}

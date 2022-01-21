package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// AddInvoiceController : Add invoice controller struct
type AddInvoiceController struct {
	svc *service.LndhubService
}

func NewAddInvoiceController(svc *service.LndhubService) *AddInvoiceController {
	return &AddInvoiceController{svc: svc}
}

// AddInvoice : Add invoice Controller
func (controller *AddInvoiceController) AddInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	type RequestBody struct {
		Amount          interface{} `json:"amt"` // amount in Satoshi
		Memo            string      `json:"memo"`
		DescriptionHash string      `json:"description_hash"`
	}

	var body RequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "Bad arguments",
		})
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "Bad arguments",
		})
	}

	amount, err := controller.svc.ParseInt(body.Amount)
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error":   true,
			"code":    8,
			"message": "Bad arguments",
		})
	}
	c.Logger().Infof("Adding invoice: user_id=%v memo=%s value=%v description_hash=%s", userID, body.Memo, amount, body.DescriptionHash)

	invoice, err := controller.svc.AddIncomingInvoice(userID, amount, body.Memo, body.DescriptionHash)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: %v", err)
		// TODO: sentry notification
		return c.JSON(http.StatusInternalServerError, nil)
	}

	var responseBody struct {
		RHash          string `json:"r_hash"`
		PaymentRequest string `json:"payment_request"`
		PayReq         string `json:"pay_req"`
	}

	responseBody.RHash = invoice.RHash
	responseBody.PaymentRequest = invoice.PaymentRequest
	responseBody.PayReq = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)
}

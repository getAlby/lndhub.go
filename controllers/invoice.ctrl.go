package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

// InvoiceController : Add invoice controller struct
type InvoiceController struct {
	svc *service.LndhubService
}

func NewInvoiceController(svc *service.LndhubService) *InvoiceController {
	return &InvoiceController{svc: svc}
}

// Invoice : Invoice Controller
func (controller *InvoiceController) Invoice(c echo.Context) error {
	user, err := controller.svc.FindUserByLogin(c.Request().Context(), c.Param("user_login"))
	if err != nil {
		c.Logger().Errorf("Failed to find user by login: login %v error %v", c.Param("user_login"), err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	var body AddInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load invoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid invoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	amount, err := controller.svc.ParseInt(body.Amount)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	c.Logger().Infof("Adding invoice: user_id=%v memo=%s value=%v description_hash=%s", user.ID, body.Memo, amount, body.DescriptionHash)

	invoice, err := controller.svc.AddIncomingInvoice(c.Request().Context(), user.ID, amount, body.Memo, body.DescriptionHash)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: %v", err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	responseBody := AddInvoiceResponseBody{}
	responseBody.RHash = invoice.RHash
	responseBody.PaymentRequest = invoice.PaymentRequest
	responseBody.PayReq = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)
}

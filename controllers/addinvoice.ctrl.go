package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
)

// AddInvoiceController : Add invoice controller struct
type AddInvoiceController struct {
	svc    *service.LndhubService
	plugin func(*models.Invoice, *service.LndhubService) (*models.Invoice, error)
}

func NewAddInvoiceController(svc *service.LndhubService) *AddInvoiceController {
	result := &AddInvoiceController{svc: svc}
	//check for plugin
	if plug, ok := svc.MiddlewarePlugins["addinvoice"]; ok {
		mwPlugin := plug.Interface().(func(in *models.Invoice, svc *service.LndhubService) (*models.Invoice, error))
		result.plugin = mwPlugin
	}

	return result
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

// AddInvoice : Add invoice Controller
func (controller *AddInvoiceController) AddInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	var body AddInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid addinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	amount, err := controller.svc.ParseInt(body.Amount)
	if err != nil {
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	c.Logger().Infof("Adding invoice: user_id=%v memo=%s value=%v description_hash=%s", userID, body.Memo, amount, body.DescriptionHash)

	invoice, err := controller.svc.AddIncomingInvoice(c.Request().Context(), userID, amount, body.Memo, body.DescriptionHash)
	if err != nil {
		c.Logger().Errorf("Error creating invoice: %v", err)
		sentry.CaptureException(err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	if controller.plugin != nil {
		invoice, err = controller.plugin(invoice, controller.svc)
		if err != nil {
			return err
		}
	}
	responseBody := AddInvoiceResponseBody{}
	responseBody.RHash = invoice.RHash
	responseBody.PaymentRequest = invoice.PaymentRequest
	responseBody.PayReq = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)
}

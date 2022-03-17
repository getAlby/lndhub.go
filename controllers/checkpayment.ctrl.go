package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// CheckPaymentController : CheckPaymentController struct
type CheckPaymentController struct {
	svc    *service.LndhubService
	plugin func(*models.Invoice, *service.LndhubService) (*models.Invoice, error)
}

type CheckPaymentResponseBody struct {
	IsPaid bool `json:"paid"`
}

func NewCheckPaymentController(svc *service.LndhubService) *CheckPaymentController {
	result := &CheckPaymentController{svc: svc}
	//check for plugin
	if plug, ok := svc.MiddlewarePlugins["checkpayment"]; ok {
		mwPlugin := plug.Interface().(func(in *models.Invoice, svc *service.LndhubService) (*models.Invoice, error))
		result.plugin = mwPlugin
	}

	return result
}

// CheckPayment : Check Payment Controller
func (controller *CheckPaymentController) CheckPayment(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	rHash := c.Param("payment_hash")

	invoice, err := controller.svc.FindInvoiceByPaymentHash(c.Request().Context(), userId, rHash)

	// Probably we did not find the invoice
	if err != nil {
		c.Logger().Errorf("Invalid checkpayment request payment_hash=%s", rHash)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	if controller.plugin != nil {
		invoice, err = controller.plugin(invoice, controller.svc)
		if err != nil {
			return err
		}
	}
	responseBody := &CheckPaymentResponseBody{}
	responseBody.IsPaid = !invoice.SettledAt.IsZero()
	return c.JSON(http.StatusOK, &responseBody)
}

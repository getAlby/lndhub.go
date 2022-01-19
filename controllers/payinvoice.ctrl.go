package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct {
	svc *lib.LndhubService
}

func NewPayInvoiceController(svc *lib.LndhubService) *PayInvoiceController {
	return &PayInvoiceController{svc: svc}
}

// PayInvoice : Pay invoice Controller
func (controller *PayInvoiceController) PayInvoice(c echo.Context) error {
	userId := c.Get("UserID").(int64)
	var reqBody struct {
		Invoice string `json:"invoice" validate:"required"`
		Amount  int    `json:"amount" validate:"omitempty,gte=0"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to bind json",
		})
	}

	if err := c.Validate(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "invalid request",
		})
	}
	//TODO json response
	return controller.svc.Payinvoice(userId, reqBody.Invoice)
}

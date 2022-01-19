package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct {
	svc *lib.LndhubService
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

	debitAccount, err := controller.svc.AccountFor(context.TODO(), "current", userId)
	if err != nil {
		return err
	}
	creditAccount, err := controller.svc.AccountFor(context.TODO(), "outgoing", userId)
	if err != nil {
		return err
	}

	entry := models.TransactionEntry{
		UserID:          userId,
		CreditAccountID: creditAccount.ID,
		DebitAccountID:  debitAccount.ID,
		Amount:          1000,
	}
	if _, err := controller.svc.DB.NewInsert().Model(&entry).Exec(context.TODO()); err != nil {
		return err
	}

	return nil
}

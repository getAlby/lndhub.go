package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
)

// PayInvoiceController : Pay invoice controller struct
type PayInvoiceController struct{}

// PayInvoice : Pay invoice Controller
func (PayInvoiceController) PayInvoice(c echo.Context) error {
	ctx := c.(*lib.LndhubService)
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

	db := ctx.DB
	debitAccount, err := ctx.User.AccountFor("current", context.TODO(), db)
	if err != nil {
		return err
	}
	creditAccount, err := ctx.User.AccountFor("outgoing", context.TODO(), db)
	if err != nil {
		return err
	}

	entry := models.TransactionEntry{
		UserID:          ctx.User.ID,
		CreditAccountID: creditAccount.ID,
		DebitAccountID:  debitAccount.ID,
		Amount:          1000,
	}
	if _, err := db.NewInsert().Model(&entry).Exec(context.TODO()); err != nil {
		return err
	}

	return nil
}

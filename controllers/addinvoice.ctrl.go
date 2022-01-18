package controllers

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib"
	"github.com/getAlby/lndhub.go/lnd"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
)

// AddInvoiceController : Add invoice controller struct
type AddInvoiceController struct{}

// AddInvoice : Add invoice Controller
func (AddInvoiceController) AddInvoice(c echo.Context) error {
	ctx := c.(*lib.LndhubContext)
	user := ctx.User

	type RequestBody struct {
		Amt             int64  `json:"amt" validate:"required"`
		Memo            string `json:"memo"`
		DescriptionHash string `json:"description_hash"`
	}

	var body RequestBody

	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to bind json, amt field with positive value is required",
		})
	}

	if err := c.Validate(&body); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "amt with positive value is required",
		})
	}

	rPreimage := makePreimageHex()

	lndProducer := lnd.NewLNDInterface(ctx)
	inv, err := lndProducer.AddInvoice(ctx.Context, body.Amt, body.Memo, rPreimage, 3600*24)
	if err != nil {
		c.Logger().Errorf("add invoice lnd client error: %v", err)
		return c.JSON(http.StatusBadGateway, nil)
	}

	db := ctx.DB

	invoice := models.Invoice{
		Type:               "",
		UserID:             user.ID,
		TransactionEntryID: 0,
		Amount:             body.Amt,
		Memo:               body.Memo,
		DescriptionHash:    body.DescriptionHash,
		PaymentRequest:     inv.PaymentRequest,
		RHash:              inv.PaymentHash,
		State:              "",
	}

	// TODO: move this to a service layer and call a method
	_, err = db.NewInsert().Model(&invoice).Exec(context.TODO())
	if err != nil {
		c.Logger().Errorf("error saving an invoice: %v", err)
		// TODO: better error handling, possibly panic and catch in an error handler
		return c.JSON(http.StatusInternalServerError, nil)
	}

	var responseBody struct {
		RHash          string `json:"r_hash"`
		PaymentRequest string `json:"payment_request"`
		PayReq         string `json:"pay_req"`
	}

	responseBody.PayReq = inv.PaymentRequest
	responseBody.PaymentRequest = inv.PaymentRequest
	responseBody.RHash = inv.PaymentHash

	return c.JSON(http.StatusOK, &responseBody)
}

const hexBytes = random.Hex

func makePreimageHex() []byte {
	b := make([]byte, 32)
	for i := range b {
		b[i] = hexBytes[rand.Intn(len(hexBytes))]
	}
	return b
}

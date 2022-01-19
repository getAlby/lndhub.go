package controllers

import (
	"math/rand"
	"net/http"

	"github.com/getAlby/lndhub.go/lib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
)

// AddInvoiceController : Add invoice controller struct
type AddInvoiceController struct {
	svc *lib.LndhubService
}

func NewAddInvoiceController(svc *lib.LndhubService) *AddInvoiceController {
	return &AddInvoiceController{svc: svc}
}

// AddInvoice : Add invoice Controller
func (controller *AddInvoiceController) AddInvoice(c echo.Context) error {
	userID := c.Get("UserID").(int64)
	type RequestBody struct {
		Amt             uint   `json:"amt" validate:"required"`
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

	invoice, err := controller.svc.AddInvoice(userID, body.Amt, body.Memo, body.DescriptionHash)
	if err != nil {
		c.Logger().Errorf("error saving an invoice: %v", err)
		return c.JSON(http.StatusInternalServerError, nil)
	}
	var responseBody struct {
		RHash          string `json:"r_hash"`
		PaymentRequest string `json:"payment_request"`
		PayReq         string `json:"pay_req"`
	}

	//TODO
	responseBody.PayReq = makePreimageHex()
	responseBody.PaymentRequest = invoice.PaymentRequest

	return c.JSON(http.StatusOK, &responseBody)
}

const hexBytes = random.Hex

func makePreimageHex() string {
	b := make([]byte, 32)
	for i := range b {
		b[i] = hexBytes[rand.Intn(len(hexBytes))]
	}
	return string(b)
}

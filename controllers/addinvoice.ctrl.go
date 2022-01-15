package controllers

import (
	"math/rand"
	"net/http"

	"github.com/bumi/lndhub.go/db/models"
	"gorm.io/gorm"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"
)

// AddInvoiceController : Add invoice controller struct
type AddInvoiceController struct{}

// AddInvoice : Add invoice Controller
func (AddInvoiceController) AddInvoice(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	userID := claims["id"].(float64)

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

	db, _ := c.Get("db").(*gorm.DB)

	invoice := models.Invoice{
		Type:               "",
		UserID:             uint(userID),
		TransactionEntryID: 0,
		Amount:             body.Amt,
		Memo:               body.Memo,
		DescriptionHash:    body.DescriptionHash,
		PaymentRequest:     "",
		RHash:              "",
		State:              "",
	}

	db.Create(&invoice)

	var responseBody struct {
		RHash          string `json:"r_hash"`
		PaymentRequest string `json:"payment_request"`
		PayReq         string `json:"pay_req"`
	}

	responseBody.PayReq = makePreimageHex()

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

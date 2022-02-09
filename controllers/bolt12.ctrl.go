package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// Bolt12Controller : Bolt12Controller struct
type Bolt12Controller struct {
	svc *service.LndhubService
}
type FetchInvoiceRequestBody struct {
	Amount int64  `json:"amt" validate:"required"` // todo: validate properly, amount not strictly needed always amount in Satoshi
	Memo   string `json:"memo"`
	Offer  string `json:"offer" validate:"required"`
}

func NewBolt12Controller(svc *service.LndhubService) *Bolt12Controller {
	return &Bolt12Controller{svc: svc}
}

// Decode : Decode handler
func (controller *Bolt12Controller) Decode(c echo.Context) error {
	offer := c.Param("offer")
	decoded, err := controller.svc.LndClient.DecodeBolt12(context.TODO(), offer)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, decoded)
}

// FetchInvoice: fetches an invoice from a bolt12 offer for a certain amount
func (controller *Bolt12Controller) FetchInvoice(c echo.Context) error {
	var body FetchInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	invoice, err := controller.svc.FetchBolt12Invoice(context.TODO(), body.Offer, body.Memo, body.Amount)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, invoice)
}

// PayOffer: fetches an invoice from a bolt12 offer for a certain amount, and pays it
func (controller *Bolt12Controller) PayOffer(c echo.Context) error {
	var body FetchInvoiceRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid fetchinvoice request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	invoice, err := controller.svc.FetchBolt12Invoice(context.TODO(), body.Offer, body.Memo, body.Amount)
	if err != nil {
		return err
	}
	result, err := controller.svc.PayBolt12Invoice(context.TODO(), invoice)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, result)
}

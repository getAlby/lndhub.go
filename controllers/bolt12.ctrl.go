package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// Bolt12Controller : Bolt12Controller struct
type Bolt12Controller struct {
	svc *service.LndhubService
}

func NewBolt12Controller(svc *service.LndhubService) *Bolt12Controller {
	return &Bolt12Controller{svc: svc}
}

// Decode : Decode handler
func (controller *Bolt12Controller) Decode(c echo.Context) error {
	offer := c.Param("offer")
	decoded, err := controller.svc.LndClient.DecodeOffer(context.TODO(), offer)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, decoded)
}

package controllers

import (
	"context"
	"net/http"

	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// GetInfoController : GetInfoController struct
type GetInfoController struct {
	svc *service.LndhubService
}

func NewGetInfoController(svc *service.LndhubService) *GetInfoController {
	return &GetInfoController{svc: svc}
}

// GetInfo : GetInfo handler
func (controller *GetInfoController) GetInfo(c echo.Context) error {

	// TODO: add some caching for this GetInfo call. No need to always hit the node
	info, err := controller.svc.GetInfo(context.TODO())
	if err != nil {
		return err
	}
	if controller.svc.Config.CustomName != "" {
		info.Alias = controller.svc.Config.CustomName
	}
	// BlueWallet right now requires a `identity_pubkey` in the response
	// https://github.com/BlueWallet/BlueWallet/blob/a28a2b96bce0bff6d1a24a951b59dc972369e490/class/wallets/lightning-custodian-wallet.js#L578
	return c.JSON(http.StatusOK, info)
}

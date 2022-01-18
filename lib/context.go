package lib

import (
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/labstack/echo/v4"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/uptrace/bun"
)

type LndhubContext struct {
	echo.Context

	DB        *bun.DB
	User      *models.User
	LndClient *lnrpc.LightningClient
}

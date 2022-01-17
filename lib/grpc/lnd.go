package grpc

import (
	"github.com/bumi/lndhub.go/lib"
	"github.com/bumi/lndhub.go/lnd"
	"github.com/labstack/echo/v4"
)

func LNDClient(lnd *lnd.LNDclient) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cc := &lib.LndhubContext{LndClient: lnd}
			return next(cc)
		}
	}
}

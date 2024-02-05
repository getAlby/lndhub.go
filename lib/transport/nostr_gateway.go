package transport

import (
	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)
// TODO learn about the strict rate limit middleware and why only
// 		a few endpoints use the logMw (could be because svc has one too).
func NostrGateway(
	svc *service.LndhubService, 
	e *echo.Echo, 
	nostrRouter *echo.Group,
) {
	nostrEventCtrl := v2controllers.NewNostrController(svc)
	e.POST("/v2/event", nostrEventCtrl.HandleNostrEvent)
}
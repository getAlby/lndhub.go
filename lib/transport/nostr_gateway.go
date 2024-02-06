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
) {
	nostrEventCtrl := v2controllers.NewNostrController(svc)
	// provides means to get pubkey without creating an event
	e.GET("/api/pubkey", nostrEventCtrl.GetServerPubkey)
	// the nostr event handler endpoint
	e.POST("/event", nostrEventCtrl.HandleNostrEvent)
}

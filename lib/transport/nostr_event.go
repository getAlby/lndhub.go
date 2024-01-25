package transport

import (
	v2controllers "github.com/getAlby/lndhub.go/controllers_v2"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

func RegisterNoStrevent(svc *service.LndhubService, e *echo.Echo, validateNostr *echo.Group) {

	nostrEventCtrl := v2controllers.NewNoStrController(svc)

	// add the endpoint to the group 
	validateNostr.POST("/v2/event", nostrEventCtrl.AddNoStrEvent)
	
}

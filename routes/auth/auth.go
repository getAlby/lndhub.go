package auth

import (
	"github.com/labstack/echo/v4"
)

// AuthRouter : AuthRouter struct
type AuthRouter struct{}

// Init : Init Router
func (ctrl AuthRouter) Init(g *echo.Group) {
	g.POST("", ctrl.Auth)
}

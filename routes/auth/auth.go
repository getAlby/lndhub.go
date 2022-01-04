package auth

import (
	"github.com/bumi/lndhub.go/lib/middlewares"
	"github.com/labstack/echo/v4"
)

// AuthRouter : AuthRouter struct
type AuthRouter struct{}

// Init : Init Router
func (ctrl AuthRouter) Init(g *echo.Group) {
	g.POST("/register", ctrl.Register)
	g.POST("/login", ctrl.Login)
	g.POST("/logout", ctrl.Logout, middlewares.Authoriszed)
}

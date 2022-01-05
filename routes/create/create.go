package create

import "github.com/labstack/echo/v4"

type CreateUserRouter struct{}

// Init : Init Router
func (ctrl CreateUserRouter) Init(g *echo.Group) {
	g.POST("", ctrl.CreateUser)
}

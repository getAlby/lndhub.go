package routes

import (
	"github.com/bumi/lndhub.go/routes/auth"
	"github.com/labstack/echo/v4"
)

// Routes : Init Routes
func Routes(g *echo.Group) {
	auth.AuthRouter{}.Init(g.Group("/auth"))
}

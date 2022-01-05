package routes

import (
	"github.com/bumi/lndhub.go/routes/addinvoice"
	"github.com/bumi/lndhub.go/routes/auth"
	"github.com/bumi/lndhub.go/routes/create"
	"github.com/labstack/echo/v4"
)

// Routes : Init Routes
func Routes(g *echo.Group) {
	auth.AuthRouter{}.Init(g.Group("/auth"))
	create.CreateUserRouter{}.Init(g.Group("/create"))
	addinvoice.AddInvoiceRouter{}.Init(g.Group("/addinvoice"))
}

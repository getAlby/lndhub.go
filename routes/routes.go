package routes

import (
	"github.com/bumi/lndhub.go/routes/addinvoice"
	"github.com/bumi/lndhub.go/routes/auth"
	"github.com/bumi/lndhub.go/routes/create"
	"github.com/labstack/echo/v4"
)

// Routes : Init Routes
func Routes(g *echo.Group) {
	g.POST("/auth", auth.AuthRouter{}.Auth)
	g.POST("/create", create.CreateUserRouter{}.CreateUser)
	g.POST("/addinvoice", addinvoice.AddInvoiceRouter{}.AddInvoice)
}

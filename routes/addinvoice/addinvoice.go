package addinvoice

import "github.com/labstack/echo/v4"

type AddInvoiceRouter struct{}

// Init : Init Router
func (ctrl AddInvoiceRouter) Init(g *echo.Group) {
	g.POST("", ctrl.AddInvoice)
}

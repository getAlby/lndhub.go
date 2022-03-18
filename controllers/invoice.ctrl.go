package controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// InvoiceController : Add invoice controller struct
type InvoiceController struct {
	svc *service.LndhubService
}

func NewInvoiceController(svc *service.LndhubService) *InvoiceController {
	return &InvoiceController{svc: svc}
}

// Invoice : Invoice Controller
func (controller *InvoiceController) Invoice(c echo.Context) error {
	user, err := controller.svc.FindUserByLogin(c.Request().Context(), c.Param("user_login"))
	if err != nil {
		c.Logger().Errorf("Failed to find user by login: login %v error %v", c.Param("user_login"), err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	return AddInvoice(c, controller.svc, user.ID)
}

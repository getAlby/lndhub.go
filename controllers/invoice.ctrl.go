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

// Invoice godoc
// @Summary      Generate a new invoice
// @Description  Returns a new bolt11 invoice for a user with given login, without an Authorization Header
// @Accept       json
// @Produce      json
// @Tags         Invoice
// @Param        user_login  path      string                 true  "User Login"
// @Param        invoice     body      AddInvoiceRequestBody  True  "Add Invoice"
// @Success      200         {object}  AddInvoiceResponseBody
// @Failure      400         {object}  responses.ErrorResponse
// @Failure      500         {object}  responses.ErrorResponse
// @Router       /invoice/{user_login} [post]
func (controller *InvoiceController) Invoice(c echo.Context) error {
	user, err := controller.svc.FindUserByLogin(c.Request().Context(), c.Param("user_login"))
	if err != nil {
		c.Logger().Errorf("Failed to find user by login: login %v error %v", c.Param("user_login"), err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	return AddInvoice(c, controller.svc, user.ID)
}

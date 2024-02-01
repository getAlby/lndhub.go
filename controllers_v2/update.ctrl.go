package v2controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// UpdateUserController : Update user controller struct
type UpdateUserController struct {
	svc *service.LndhubService
}

func NewUpdateUserController(svc *service.LndhubService) *UpdateUserController {
	return &UpdateUserController{svc: svc}
}

type UpdateUserResponseBody struct {
	Pubkey      string `json:"pubkey"`
	Deactivated bool   `json:"deactivated"`
	Deleted     bool   `json:"deleted"`
	ID          int64  `json:"id"`
}
type UpdateUserRequestBody struct {
	Pubkey      *string `json:"pubkey,omitempty"`
	Password    *string `json:"password,omitempty"`
	Deactivated *bool   `json:"deactivated,omitempty"`
	Deleted     *bool   `json:"deleted,omitempty"`
	ID          int64   `json:"id" validate:"required"`
}

// UpdateUser godoc
// @Summary      Update an account
// @Description  Update an account with a new a login, password and activation status. Requires Authorization header with admin token.
// @Accept       json
// @Produce      json
// @Tags         Account
// @Param        account  body      UpdateUserRequestBody  false  "Update User"
// @Success      200      {object}  UpdateUserResponseBody
// @Failure      400      {object}  responses.ErrorResponse
// @Failure      500      {object}  responses.ErrorResponse
// @Router       /v2/admin/users [put]
func (controller *UpdateUserController) UpdateUser(c echo.Context) error {

	var body UpdateUserRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load update user request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid update user request body error: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	user, err := controller.svc.UpdateUser(c.Request().Context(), body.ID, body.Pubkey, body.Password, body.Deactivated)
	if err != nil {
		c.Logger().Errorf("Failed to update user: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	var ResponseBody UpdateUserResponseBody
	ResponseBody.Pubkey = user.Pubkey
	ResponseBody.Deactivated = user.Deactivated
	ResponseBody.Deleted = user.Deleted
	ResponseBody.ID = user.ID

	return c.JSON(http.StatusOK, &ResponseBody)
}

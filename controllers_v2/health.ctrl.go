package v2controllers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthController struct {
}

func NewHealthController() *HealthController {
	return &HealthController{}
}

type HealthResponse struct {
	Result string `json:"result"`
}

// Health godoc
// @Summary      Check system health
// @Description  Check system health
// @Accept       json
// @Produce      json
// @Tags         Account
// @Success      200  {object}  HealthResponse
// @Failure      400  {object}  responses.ErrorResponse
// @Failure      500  {object}  responses.ErrorResponse
// @Router       /v2/balance [get]
// @Security     OAuth2Password
func (controller *HealthController) Check(c echo.Context) error {
	return c.JSON(http.StatusOK, &HealthResponse{
		Result: "OK",
	})
}

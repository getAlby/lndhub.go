package v2controllers

import (
	"net/http"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
)

// NostrController : Add NoStr Event controller struct
type NostrController struct {
	svc *service.LndhubService
}

func NewNostrController(svc *service.LndhubService) *NostrController {
	return &NostrController{svc: svc}
}


type AddNoStrResponseBody struct  {
	ID        string `json:"ID"`
	Pubkey    string `json:"Pubkey"`
	Kind      int    `json:"kind"`
	Ta        string `json:"Ta"`
	Amt       float64   `json:"Amt"`
	Addr      string   `json:"addr"`
	Fee       float64  `json:"fee"`
	Content   string `json:"Content"`
	Sig       string `json:"Sig"`
}

// AddNoStrEvent godoc
// @Summary      Validate NostEvent Payload
// @Description  Returns a new AddNoStrResponseBody
// @Accept       json
// @Produce      json
// @Tags         NoStrEvent
// @Body      AddNoStrResponseBody  True  "Add NoStr Event"
// @Success      200      {object}  AddNoStrResponseBody
// @Failure      400      {object}  responses.ErrorResponse
// @Failure      500      {object}  responses.ErrorResponse
// @Router       /v2/event [post]
func (controller *NostrController) AddNostrEvent(c echo.Context) error {
	
	var body AddNoStrResponseBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load AddNoStrEvent request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid AddNoStrEvent request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}


	responseBody := AddNoStrResponseBody{
		ID: body.ID,
		Pubkey: body.Pubkey,
		Content:  body.Content,
		Kind: body.Kind,
		Amt: body.Amt,
		Fee: body.Fee,
		Ta: body.Ta,
		Addr: body.Addr,
		Sig: body.Sig,
	}

	return c.JSON(http.StatusOK, &responseBody)
}


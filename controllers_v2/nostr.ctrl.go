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

type EventRequestBody struct {
	ID        string            `json:"ID"`
	Pubkey    string            `json:"Pubkey"`
	CreatedAt int64             `json:"CreatedAt"`
	Kind      int               `json:"kind"`
	Tags      [][]interface{}   `json:"tags"`
	Content   string            `json:"Content"`
	Sig       string            `json:"Sig"`
	Addr      string            `json:"Addr"`
	Fee       float64           `json:"Fee"`
}

type CreateUserEventResponseBody struct {
	// internal tahub user id
	ID     int64 `json:"id"`
	// nostr public key, discovered via the event
	Pubkey string `json:"pubkey"`
}

func (controller *NostrController) AddNostrEvent(c echo.Context) error {
	
	var body EventRequestBody

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load AddNostrEvent request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid AddNostrEvent request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	// handle create user event - can assume valid thanks to middleware
	if body.Content == "TAHUB_CREATE_USER" {
		user, err := controller.svc.CreateUser(c.Request().Context(), body.Pubkey)
		if err != nil {
			// create user error response
			c.Logger().Errorf("Failed to create user via Nostr event: %v", err)
			return c.JSON(http.StatusInternalServerError, responses.GeneralServerError)
		}
		// create user success response
		var ResponseBody CreateUserEventResponseBody
		ResponseBody.ID = user.ID
		ResponseBody.Pubkey = user.Pubkey

		return c.JSON(http.StatusOK, &ResponseBody)
	} else {
		// TODO handle next events
		return c.JSON(http.StatusBadRequest, responses.UnimplementedError)
	}
}


// TODO - record events in the database, requires a model
// type Event struct  {
// 	ID        string    `json:"ID"`
// 	Pubkey    string    `json:"Pubkey"`
// 	Kind      int       `json:"kind"`
// 	Ta        string    `json:"Ta"`
// 	Amt       float64   `json:"Amt"`
// 	Addr      string    `json:"addr"`
// 	Fee       float64   `json:"fee"`
// 	Content   string    `json:"Content"`
// 	Sig       string    `json:"Sig"`
// }


// NostrEventBody godoc
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
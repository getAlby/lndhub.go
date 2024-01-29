package v2controllers

import (
	"net/http"
	"github.com/nbd-wtf/go-nostr"
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

// type EventRequestBody struct {
// 	ID        string            `json:"id"`
// 	Pubkey    string            `json:"pubkey"`
// 	CreatedAt int64             `json:"created_at"`
// 	Kind      int               `json:"kind"`
// 	Tags      [][]interface{}   `json:"tags"`
// 	Content   string            `json:"Content"`
// 	Sig       string            `json:"Sig"`
// }

type CreateUserEventResponseBody struct {
	// internal tahub user id
	ID     int64 `json:"id"`
	// nostr public key, discovered via the event
	Pubkey string `json:"pubkey"`
}

func (controller *NostrController) HandleNostrEvent(c echo.Context) error {
	
	var body nostr.Event

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load Nostr Event request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid Nostr Event request body: %v", err)
		return c.JSON(http.StatusBadRequest, responses.BadArgumentsError)
	}
	// check signature
	if result, err := body.CheckSignature(); (err != nil || !result) {
		c.Logger().Errorf("Signature is not valid for the event... Consider monitoring this user if issue persists: %v", err)
		return c.JSON(http.StatusUnauthorized, responses.BadAuthError)
	}
	// handle create user event - can assume valid thanks to middleware
	if body.Content == "TAHUB_CREATE_USER" {
		user, err := controller.svc.CreateUser(c.Request().Context(), body.PubKey)
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

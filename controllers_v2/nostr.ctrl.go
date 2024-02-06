package v2controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip19"
)

// NostrController : Add NoStr Event controller struct
type NostrController struct {
	svc *service.LndhubService
	responder responses.RelayResponder
}

func NewNostrController(svc *service.LndhubService) *NostrController {
	return &NostrController{svc: svc, responder: responses.RelayResponder{}}
}


// A utility endpoint to recover the server pubkey w/o creating a nostr event
func (controller *NostrController) GetServerPubkey(c echo.Context) error {
	res, err := controller.HandleGetPublicKey()
	if err != nil {
		c.Logger().Errorf("Failed to handle / encode public key: %v", err)
		return c.JSON(http.StatusInternalServerError, responses.NostrServerError)
	}

	return c.JSON(http.StatusOK, &res)
}

func (controller *NostrController) HandleNostrEvent(c echo.Context) error {
	// The main nostr event handler
	var body nostr.Event

	if err := c.Bind(&body); err != nil {
		c.Logger().Errorf("Failed to load Nostr Event request body: %v", err)
		return controller.responder.NostrErrorResponse(c, responses.BadArgumentsError.Message)
	}

	if err := c.Validate(&body); err != nil {
		c.Logger().Errorf("Invalid Nostr Event request body: %v", err)
		return controller.responder.NostrErrorResponse(c, responses.BadArgumentsError.Message)
	}
	// check signature
	if result, err := body.CheckSignature(); (err != nil || !result) {
		c.Logger().Errorf("Signature is not valid for the event... Consider monitoring this user if issue persists: %v", err)
		return controller.responder.NostrErrorResponse(c, responses.BadAuthError.Message)
	}
	// TODO add NIP4 decoding here

	// call our payload validator 
	result, decodedPayload, err := controller.svc.CheckEvent(body)
	if err != nil || !result {
		c.Logger().Errorf("Invalid Nostr Event content: %v", err)
		return controller.responder.NostrErrorResponse(c, responses.InvalidTahubContentError.Message)
	}
	// Split event content
	data := strings.Split(decodedPayload.Content, ":")
	// handle create user event - can assume valid thanks to middleware
	if data[0] == "TAHUB_CREATE_USER" {
		// TODO determine if a check against config is required
		// 		in Tahub's case: https://github.com/nostrassets/Tahub.go/blob/a798601f63d5847b045360e45e8090081bb4cd85/lib/transport/v2_endpoints.go#L12
		// check if user exists
		existingUser, err := controller.svc.FindUserByPubkey(c.Request().Context(), body.PubKey)
		// check if user was found
		if existingUser.ID > 0 {
			c.Logger().Errorf("Cannot create user that has already registered this pubkey")
			return controller.responder.CreateUserOk(c, body, existingUser.ID, true, "this pubkey has already been registered.")
		}
		// confirm no error occurred in checking if the user exists
		if err != nil {
			msg := err.Error()
			// TODO consider this and try to make more robust
			if msg != "sql: now rows in result set" {
				c.Logger().Info("Error is related to no results in the dataset, which is acceptable.")
			} else {
				c.Logger().Errorf("Unable to verify the pubkey has not already been registered: %v", err)
				return controller.responder.CreateUserOk(c, body, 0, true, "failed to check pubkey.")
			}
		}
		// create the user, by public key
		user, err := controller.svc.CreateUser(c.Request().Context(), body.PubKey)
		if err != nil {
			// create user error response
			c.Logger().Errorf("Failed to create user via Nostr event: %v", err)
			return controller.responder.CreateUserOk(c, body, 0, true, "failed to insert user into database.")
		}
		// create user success response
		return controller.responder.CreateUserOk(c, body, user.ID, false, "")
	} else if data[0] == "TAHUB_GET_SERVER_PUBKEY" {
		// get server npub
		res, err := controller.HandleGetPublicKey()
		if err != nil {
			c.Logger().Errorf("Failed to handle / encode public key: %v", err)
			return controller.responder.GetServerPubkeyOk(c, body, "", true, responses.NostrServerError.Message)
		}
		// return server npub
		return controller.responder.GetServerPubkeyOk(c, body, res.TahubPubkeyHex, false, "")

	} else if data[0] == "TAHUB_GET_UNIVERSE_ASSETS" {
		// get universe known assets 
		msg, status := controller.svc.GetUniverseAssets(c.Request().Context())
		return controller.responder.GenericOk(c, body, msg, status)
	} else if data[0] == "TAHUB_GET_RCV_ADDR" {
		// given an asset_id and amt, return the address

		// these values are prevalidated by CheckEvent
		assetId := data[1]
		amt, err := strconv.ParseUint(data[2], 10, 64)
		if err != nil {
			c.Logger().Errorf("Failed to parse amt field in content: %v", err)
			return controller.responder.NostrErrorResponse(c, responses.GeneralServerError.Message)
		}
		msg, status := controller.svc.GetAddressByAssetId(c.Request().Context(), assetId, amt)
		return controller.responder.GenericOk(c, body, msg, status)
	} else {
		// TODO handle next events
		return controller.responder.NostrErrorResponse(c, "unimplemented.")
	}
}

func (controller *NostrController) HandleGetPublicKey() (responses.GetServerPubkeyResponseBody, error) {
	var ResponseBody responses.GetServerPubkeyResponseBody
	ResponseBody.TahubPubkeyHex = controller.svc.Config.TahubPublicKey
	npub, err := nip19.EncodePublicKey(controller.svc.Config.TahubPublicKey)
	// TODO improve this
	if err != nil {
		return ResponseBody, err
	}
	ResponseBody.TahubNpub = npub
	return ResponseBody, nil
}
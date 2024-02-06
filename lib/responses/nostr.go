package responses

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nbd-wtf/go-nostr"
)

type RelayResponder struct {}
// create a parent or interface to type the nostr responses together
// in order to provide a type and new parameter to a standardized 
// relay-compatible responder
func (responder *RelayResponder) NostrErrorResponse(c echo.Context, errMsg string) error {
	msg := fmt.Sprintf("error: %s", errMsg)
	res := []interface{}{"OK", -1, false, msg}

	return c.JSON(http.StatusOK, res)
}

func (responder *RelayResponder) CreateUserOk(c echo.Context, event nostr.Event, userId int64, isError bool, errMsg string) error {
	// TODO these messages should end up in a central location
	var status = false
	var msg = fmt.Sprintf("error: %s", errMsg)

	if !isError {
		msg = fmt.Sprintf("userid: %d", userId)
		status = true
	}
	
	res := []interface{} {"OK", event.ID, status, msg}
	return c.JSON(http.StatusOK, res)
}

func (responder *RelayResponder) GetServerPubkeyOk(c echo.Context, event nostr.Event, serverNpub string, isError bool, errMsg string) error {
	// TODO these messages should end up in a central location
	var status = false
	var msg = fmt.Sprintf("error: %s", errMsg)

	if !isError {
		msg = fmt.Sprintf("pubkey: %s", serverNpub)
		status = true
	}

	res := []interface{} {"OK", event.ID, status, msg}
	return c.JSON(http.StatusOK, res)
}

func (responder *RelayResponder) GenericOk(c echo.Context, event nostr.Event, msg string, status bool) error {
	// TODO these messages should end up in a central location
	res := []interface{} {"OK", event.ID, status, msg}
	return c.JSON(http.StatusOK, res)
}


type CreateUserEventResponseBody struct {
	// internal tahub user id
	ID     int64 `json:"id"`
	// nostr public key, discovered via the event
	Pubkey string `json:"pubkey"`
}

type GetServerPubkeyResponseBody struct {
	TahubPubkeyHex   string `json:"tahub_pubkey"`
	TahubNpub        string `json:"tahub_npub"`
}
package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/responses"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip19"
)

func (svc *LndhubService) EventHandler(ctx context.Context, payload nostr.Event) error {
	// check sig
	if result, err := payload.CheckSignature(); (err != nil || !result) {
		svc.Logger.Errorf("Signature is not valid for the event... Consider monitoring this user if issue persists: %v", err)
		// TODO further evaluate response, don't need to break routine
		return nil
	}
	// validate and decode
	result, decoded, err := svc.CheckEvent(payload)
	if err != nil || !result {
		svc.Logger.Errorf("Invalid Nostr Event content: %v", err)
		// TODO further evaluate response, don't need to break routine
		return nil
	}
	// insert encoded
	status, err := svc.InsertEvent(ctx, payload)
	if err != nil || !status {
		svc.Logger.Errorf("Failed to insert nostr event into db.")
		return nil
	}
	// Split event content
	data := strings.Split(decoded.Content, ":")
	// handle create user event - can assume valid thanks to middleware
	if data[0] == "TAHUB_CREATE_USER" {
		// TODO determine if a check against config is required
		// 		in Tahub's case: https://github.com/nostrassets/Tahub.go/blob/a798601f63d5847b045360e45e8090081bb4cd85/lib/transport/v2_endpoints.go#L12
		// check if user exists
		existingUser, err := svc.FindUserByPubkey(ctx, decoded.PubKey)
		// check if user was found
		if existingUser.ID > 0 {
			svc.Logger.Errorf("Cannot create user that has already registered this pubkey")

			svc.RespondToNip4(ctx, "error: exists", true, decoded.PubKey, decoded.ID,)
			// TODO further evaluate response, don't need to break routine
 			return nil
		}
		// confirm no error occurred in checking if the user exists
		if err != nil {
			msg := err.Error()
			// TODO consider this and try to make more robust
			if msg != "sql: now rows in result set" {
				svc.Logger.Info("Error is related to no results in the dataset, which is acceptable.")
			} else {
				svc.Logger.Errorf("Unable to verify the pubkey has not already been registered: %v", err)
				// TODO further evaluate response, don't need to break routine
				return nil
			}
		}
		// create the user, by public key
		user, err := svc.CreateUser(ctx, decoded.PubKey)
		if err != nil {
			// create user error response
			svc.Logger.Errorf("Failed to create user via Nostr event: %v", err)
			// TODO further evaluate response, don't need to break routine
			return nil
		}
		// create user success response
		msg := fmt.Sprintf("userid: %d", user.ID)
		svc.RespondToNip4(ctx, msg, false, decoded.PubKey, decoded.ID)

	} else if data[0] == "TAHUB_GET_SERVER_PUBKEY" {
		// get server npub
		res, err := svc.HandleGetPublicKey()
		if err != nil {
			svc.Logger.Errorf("Failed to handle / encode public key: %v", err)
			// TODO further evaluate response, don't need to break routine
			return nil
		}
		// return server npub
		// TODO respond with success
		msg := fmt.Sprintf("pubkey: %s", res.TahubPubkeyHex)
		svc.RespondToNip4(ctx, msg, false, decoded.PubKey, decoded.ID)
	} else if data[0] == "TAHUB_GET_UNIVERSE_ASSETS" {
		// get universe known assets 
		msg, status := svc.GetUniverseAssets(ctx)
		if !status {
			svc.Logger.Errorf("Failed to get universe assets from tapd: %s", msg)
			// TODO further evaluate response, don't need to break routine
			return nil
		}
		// TODO respond with success
		svc.RespondToNip4(ctx, msg, false, decoded.PubKey, decoded.ID)
	} else if data[0] == "TAHUB_GET_RCV_ADDR" {
		// given an asset_id and amt, return the address

		// these values are prevalidated by CheckEvent
		assetId := data[1]
		amt, err := strconv.ParseUint(data[2], 10, 64)
		if err != nil {
			svc.Logger.Errorf("Failed to parse amt field in content: %v", err)
			// TODO further evaluate response, don't need to break routine
			return nil
		}
		msg, status := svc.GetAddressByAssetId(ctx, assetId, amt)
		if !status {
			svc.Logger.Errorf("Failed to get rcv address for asset from tapd: %s", msg)
			// TODO further evaluate response, don't need to break routine
			return nil
		}
		// TODO respond with success
		svc.RespondToNip4(ctx, msg, false, decoded.PubKey, decoded.ID)
	} else {
		// catch all - unimplemented
		// TODO further evaluate response, don't need to break routine
		return nil
	}
	return nil
}

func (svc *LndhubService) RespondToNip4(ctx context.Context, rawContent string, errored bool, userPubkey string, replyToEventId string) (map[string]string, bool) {
	// responseContent collection
	responses := make(map[string]string)
	// default content
	var responseContent = rawContent
	// default status, set to true if additional error occurs
	errProcessing := errored
	// check for duplicate event
	var existing = models.Event{}
	existingEventQuery := svc.DB.NewSelect().Model(&existing).Where("event_id = ?", replyToEventId)
	err := existingEventQuery.Scan(context.Background())
	if err != nil || existing.EventID == replyToEventId {
		svc.Logger.Errorf("Duplicate event found.")
		responseContent = "tahuberror: dup event"
		// add to responses map
		responses["eventseen"] = responseContent
		errProcessing = true
		return responses, errProcessing
	}
	resp := nostr.Event{}
	resp.CreatedAt = nostr.Now()
	resp.PubKey = svc.Config.TahubPublicKey
	resp.Kind = nostr.KindEncryptedDirectMessage
	// TODO encrypt content
	sharedSecret, err := nip04.ComputeSharedSecret(userPubkey, svc.Config.TahubPrivateKey)
	if err != nil {
		svc.Logger.Errorf("Failed to compute shared secret for response to NIP4 dm: %v", err)
		responseContent = "tahuberror: auth, couldnt compute shared secret to respond"
		// add to responses map
		responses["nip4"] = responseContent
		errProcessing = true
		return responses, errProcessing
	}
	encryptedContent, err := nip04.Encrypt(responseContent, sharedSecret)
	if err != nil {
		svc.Logger.Errorf("Generated shared secret but failed to encrypt: %v", err)
		responseContent = "tahuberror: auth, failed to encrypt after computing shared secret"
		// add to responses map
		responses["nip4"] = responseContent
		errProcessing = true
		return responses, errProcessing
	}

	// encrypt response
	resp.Content = encryptedContent
	// make tags
	pTag := []string{"p", userPubkey}
	eTag := []string{"e", replyToEventId}

	resp.Tags = nostr.Tags{pTag, eTag}
	// sign event (handles ID and signature)
	resp.Sign(svc.Config.TahubPrivateKey)
	// broadcast TODO address warning / create type for URL
	for _, url := range svc.Config.RelayURI {
		broadcastCtx := context.WithValue(context.Background(), "url", url)
		conn, e := nostr.RelayConnect(broadcastCtx, url)
		if e != nil {
			// failed to connect to relay
			svc.Logger.Errorf("CRITICAL: failed to connect to relay while responding to event %s: %v", replyToEventId, e)
			errProcessing = true
			responseContent = "tahuberror: failed to connect to relay."
			// add to responses map
			responses[url] = responseContent
			continue
		}
		// attempt publish to relay
		publishedErr := conn.Publish(ctx, resp)

		if publishedErr != nil {
			// failed to publish event to relay
			svc.Logger.Errorf("CRITICAL: failed to publish to relay while responding to event %s: %v", replyToEventId, e)
			errProcessing = true
			responseContent = "tahuberror: failed to broadcast event to relay."
			// add to responses map
			responses[url] = responseContent

			continue
		}
		// broadcast to relay successful
		svc.Logger.Infof("Successfully broadcasted response to event %s to relay %s", replyToEventId, url)
		// add to responses map
		responses[url] = "broadcast"
		//return responseContent, errProcessing
	}
	return responses, errProcessing
}

func (svc *LndhubService) InsertEvent(ctx context.Context, payload nostr.Event) (success bool, err error) {
	// TODO look for better way to do this
	eventData := models.Event{
		EventID: payload.ID,
		FromPubkey: payload.PubKey,
		Kind: int64(payload.Kind),
		Content: payload.Content,
		CreatedAt: payload.CreatedAt.Time().Unix(),
	}
	
	// make insert TODO add logging
	_, err = svc.DB.NewInsert().Model(&eventData).Exec(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (svc *LndhubService) HandleGetPublicKey() (responses.GetServerPubkeyResponseBody, error) {
	var ResponseBody responses.GetServerPubkeyResponseBody
	ResponseBody.TahubPubkeyHex = svc.Config.TahubPublicKey
	npub, err := nip19.EncodePublicKey(svc.Config.TahubPublicKey)
	// TODO improve this
	if err != nil {
		return ResponseBody, err
	}
	ResponseBody.TahubNpub = npub
	return ResponseBody, nil
}
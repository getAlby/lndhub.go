package service

import (
	"context"

	"time"
	"github.com/nbd-wtf/go-nostr"
	//"github.com/nbd-wtf/go-nostr/nip19"
)
func (svc *LndhubService) StartRelayRoutine(ctx context.Context) (err error) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	relay, err := nostr.RelayConnect(bgCtx, svc.Config.RelayURI)

	if err != nil {
		// we need to restart on error of this routine
		return err
	}
	// create NIP 4 filter
	var filters nostr.Filters
	t := make(map[string][]string)
	// add p tag for public key
	t["p"] = []string{svc.Config.TahubPublicKey}
	filters = []nostr.Filter{{
		Kinds: []int{nostr.KindEncryptedDirectMessage},
		Tags: t,
	}}
	// create sub
	sub, _ := relay.Subscribe(ctx, filters)
	// collect errored events 
	//errEvents := make([]nostr.Event, 0)

	go func() {
		<-sub.EndOfStoredEvents
		cancel()
	}()
	// scan events
	for ev := range sub.Events {
		// append to event collection
		//errEvents = append(errEvents, *ev)
		// handle event
		err := svc.EventHandler(ctx, *ev)
		if err != nil && err != context.Canceled {
			return err
		}
	}
	return nil
}

func (svc *LndhubService) StartInvoiceRoutine(ctx context.Context) (err error) {
	if svc.RabbitMQClient != nil {
		err = svc.RabbitMQClient.SubscribeToLndInvoices(ctx, svc.ProcessInvoiceUpdate)
		if err != nil && err != context.Canceled {
			return err
		}

		return nil
	} else {
		err = svc.InvoiceUpdateSubscription(ctx)
		if err != nil && err != context.Canceled {
			// in case of an error in this routine, we want to restart LNDhub
			return err
		}

		return nil
	}
}

func (svc *LndhubService) StartPendingPaymentRoutine(ctx context.Context) (err error) {
	if svc.RabbitMQClient != nil {
		return svc.RabbitMQClient.FinalizeInitializedPayments(ctx, svc)
	} else {
		pending, err := svc.GetAllPendingPayments(ctx)
		if err != nil {
			return err
		}
		svc.Logger.Infof("Found %d pending payments", len(pending))
		return svc.CheckPendingOutgoingPayments(ctx, pending)
	}
}

package main

import (
	"context"

	"github.com/getAlby/lndhub.go/lib/service"
)

func StartInvoiceRoutine(svc *service.LndhubService, backGroundCtx context.Context) (err error) {
	if svc.RabbitMQClient != nil {
		err = svc.RabbitMQClient.SubscribeToLndInvoices(backGroundCtx, svc.ProcessInvoiceUpdate)
		if err != nil && err != context.Canceled {
			return err
		}

		return nil
	} else {
		err = svc.InvoiceUpdateSubscription(backGroundCtx)
		if err != nil && err != context.Canceled {
			// in case of an error in this routine, we want to restart LNDhub
			return err
		}

		return nil
	}
}

func StartPendingPaymentRoutine(svc *service.LndhubService, backGroundCtx context.Context) (err error) {
	if svc.RabbitMQClient != nil {
		return svc.RabbitMQClient.FinalizeInitializedPayments(backGroundCtx, svc)
	} else {
		return svc.CheckAllPendingOutgoingPayments(backGroundCtx)
	}
}

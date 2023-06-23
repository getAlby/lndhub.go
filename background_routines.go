package main

import (
	"context"
	"fmt"

	"github.com/getAlby/lndhub.go/lib/service"
)

func StartInvoiceRoutine(svc *service.LndhubService, backGroundCtx context.Context) (err error) {
	switch svc.Config.SubscriptionConsumerType {
	case "rabbitmq":
		err = svc.RabbitMQClient.SubscribeToLndInvoices(backGroundCtx, svc.ProcessInvoiceUpdate)
		if err != nil && err != context.Canceled {
			return err
		}

	case "grpc":
		err = svc.InvoiceUpdateSubscription(backGroundCtx)
		if err != nil && err != context.Canceled {
			return err
		}

	default:
		return fmt.Errorf("Unrecognized subscription consumer type %s", svc.Config.SubscriptionConsumerType)
	}
	return nil
}

func StartPendingPaymentRoutine(svc *service.LndhubService, backGroundCtx context.Context) (err error) {
	switch svc.Config.FinalizePendingPaymentsWith {
	case "rabbitmq":
		return svc.RabbitMQClient.FinalizeInitializedPayments(backGroundCtx, svc)
	default:
		return svc.CheckAllPendingOutgoingPayments(backGroundCtx)
	}
}

package service

import (
	"context"
)

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

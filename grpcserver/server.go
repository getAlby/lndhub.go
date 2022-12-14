package grpcserver

import (
	"context"
	"strconv"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lib/service"
	"github.com/getAlby/lndhub.go/lndhubrpc"
	pb "github.com/getAlby/lndhub.go/lndhubrpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// server is used to implement helloworld.GreeterServer.
type Server struct {
	pb.UnimplementedInvoiceSubscriptionServer
	svc              *service.LndhubService
	incomingInvoices chan models.Invoice
	ctx              context.Context
}

func NewGrpcServer(svc *service.LndhubService, ctx context.Context) (*Server, error) {
	incomingInvoices := make(chan models.Invoice)
	_, err := svc.InvoicePubSub.Subscribe(common.InvoiceTypeIncoming, incomingInvoices)
	if err != nil {
		return nil, err
	}
	return &Server{
		svc:              svc,
		incomingInvoices: incomingInvoices,
		ctx:              ctx,
	}, nil
}

func (s *Server) SubsribeInvoices(req *lndhubrpc.SubsribeInvoicesRequest, srv lndhubrpc.InvoiceSubscription_SubsribeInvoicesServer) error {
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case inv := <-s.incomingInvoices:
			customRecords := []*pb.Invoice_CustomRecords{}
			for key, value := range inv.DestinationCustomRecords {
				customRecords = append(customRecords, &pb.Invoice_CustomRecords{
					//todo: fix types
					Key:   strconv.Itoa(int(key)),
					Value: string(value),
				})
			}
			srv.Send(&pb.Invoice{
				Id:                   uint32(inv.ID),
				Type:                 inv.Type,
				UserId:               uint32(inv.UserID),
				Amount:               uint32(inv.Amount),
				Fee:                  uint32(inv.Fee),
				Memo:                 inv.Memo,
				DescriptionHash:      inv.DescriptionHash,
				PaymentRequest:       inv.PaymentRequest,
				DestinationPubkeyHex: inv.DestinationPubkeyHex,
				CustomRecords:        customRecords,
				RHash:                inv.RHash,
				Preimage:             inv.Preimage,
				Keysend:              inv.Keysend,
				State:                inv.State,
				CreatedAt:            timestamppb.New(inv.CreatedAt),
				ExpiresAt:            timestamppb.New(inv.ExpiresAt.Time),
				UpdatedAt:            timestamppb.New(inv.UpdatedAt.Time),
				SettledAt:            timestamppb.New(inv.SettledAt.Time),
			})
		}
	}
}

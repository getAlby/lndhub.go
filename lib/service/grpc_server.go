package service

import (
	"context"
	"fmt"
	"net"

	"github.com/getAlby/lndhub.go/common"
	"github.com/getAlby/lndhub.go/db/models"
	"github.com/getAlby/lndhub.go/lndhubrpc"
	pb "github.com/getAlby/lndhub.go/lndhubrpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (svc *LndhubService) StartGrpcServer(ctx context.Context) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", svc.Config.GRPCPort))
	if err != nil {
		svc.Logger.Fatalf("Failed to start grpc server: %v", err)
	}
	s := grpc.NewServer()
	grpcServer, err := NewGrpcServer(svc, ctx)
	if err != nil {
		svc.Logger.Fatalf("Failed to init grpc server, %s", err.Error())
	}
	lndhubrpc.RegisterInvoiceSubscriptionServer(s, grpcServer)
	svc.Logger.Infof("gRPC server started at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		svc.Logger.Fatalf("failed to serve: %v", err)
	}
}

// server is used to implement helloworld.GreeterServer.
type Server struct {
	pb.UnimplementedInvoiceSubscriptionServer
	svc *LndhubService
	ctx context.Context
}

func NewGrpcServer(svc *LndhubService, ctx context.Context) (*Server, error) {
	return &Server{
		svc: svc,
		ctx: ctx,
	}, nil
}

func (s *Server) SubsribeInvoices(req *lndhubrpc.SubsribeInvoicesRequest, srv lndhubrpc.InvoiceSubscription_SubsribeInvoicesServer) error {
	incomingInvoices := make(chan models.Invoice)
	_, err := s.svc.InvoicePubSub.Subscribe(common.InvoiceTypeIncoming, incomingInvoices)
	if err != nil {
		return err
	}
	alreadySeenId := int64(-1)
	if req.FromId != nil {
		//look up all settled incoming invoices from a certain id
		//and return them first
		invoices := []models.Invoice{}
		err := s.svc.DB.NewSelect().Model(&invoices).Where("state = 'settled'").Where("type = 'incoming'").Where("id > ?", *req.FromId).OrderExpr("id ASC").Scan(s.ctx)
		if err != nil {
			return err
		}
		//add this so we can avoid duplicates in case of a race condition:
		//when an invoice is settled in the time between the execution of
		//"InvoicePubSub.Subscribe" and the SQL query execution by the db
		//it can be both in the "incomingInvoices" channel and the SQL result set.
		if len(invoices) != 0 {
			alreadySeenId = int64(invoices[len(invoices)-1].ID)
		}
		for _, inv := range invoices {
			srv.Send(convertInvoice(inv))
		}
	}
	for {
		select {
		case <-s.ctx.Done():
			return nil
		case inv := <-incomingInvoices:
			//in case we've already send it over
			if inv.ID > alreadySeenId {
				srv.Send(convertInvoice(inv))
			}
		}
	}
}

func convertInvoice(inv models.Invoice) *pb.Invoice {
	customRecords := []*pb.Invoice_CustomRecords{}
	for key, value := range inv.DestinationCustomRecords {
		customRecords = append(customRecords, &pb.Invoice_CustomRecords{
			//todo: fix types
			Key:   key,
			Value: value,
		})
	}
	return &pb.Invoice{
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
	}
}

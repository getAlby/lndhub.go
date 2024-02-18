package service

import (
	"context"
	"github.com/getAlby/lndhub.go/db/models"
)

func (svc *LndhubService) GetRelays(ctx context.Context) ([]models.Relay, error) {
	relay := []models.Relay{}

	err := svc.DB.NewSelect().Model(&relay).Scan(ctx)
	return relay, err
}

package service

import (
	"context"
	"github.com/getAlby/lndhub.go/db/models"
)

func (svc *LndhubService) GetRelays(ctx context.Context) ([]models.Relay, error) {
	relay := []models.Relay{}

	err := svc.DB.NewSelect().Model(&relay).Distinct().Limit(5).Relation("Filter").Scan(ctx)
	return relay, err
}

func (svc *LndhubService) FindRelay(ctx context.Context, relayUri string) (*models.Relay, error) {
	var relay models.Relay

	err := svc.DB.NewSelect().Model(&relay).Where("uri = ?", relayUri).Limit(1).Relation("Filter").Scan(ctx)
	if err != nil {
		return &relay, err
	}
	return &relay, nil
}

func (svc *LndhubService) UpdateRelay(ctx context.Context, relayUri string, lastSeen int64) (relay *models.Relay, err error) {
	relay, err = svc.FindRelay(ctx, relayUri)
	if err != nil {
		return nil, err
	}
	var filter = relay.Filter
	// TODO ensure that this does not overwrite other Filter fields to null
	filter.LastEventSeen = lastSeen
	_, err = svc.DB.NewUpdate().Model(filter).WherePK().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return relay, nil
}

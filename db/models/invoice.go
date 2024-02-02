package models

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// Invoice : Invoice Model
type Invoice struct {
	ID                       int64             `json:"id" bun:",pk,autoincrement"`
	Type                     string            `json:"type" validate:"required"`
	UserID                   int64             `json:"user_id" validate:"required"`
	User                     *User             `json:"-" bun:"rel:belongs-to,join:user_id=id"`
	AssetID                  int64             `json:"asset_id" validate:"required"`
	Asset                    *Asset            `json:"-" bun:"rel:has-one,join:asset_id=id"`
	Amount                   int64             `json:"amount" validate:"gte=0"`
	Fee                      int64             `json:"fee"`
	ServiceFee               int64             `json:"service_fee"`
	RoutingFee               int64             `json:"routing_fee"`
	Memo                     string            `json:"memo" bun:",nullzero"`
	DescriptionHash          string            `json:"description_hash,omitempty" bun:",nullzero"`
	PaymentRequest           string            `json:"payment_request" bun:",nullzero"`
	DestinationPubkeyHex     string            `json:"destination_pubkey_hex" bun:",notnull"`
	DestinationCustomRecords map[uint64][]byte `json:"custom_records,omitempty"`
	RHash                    string            `json:"r_hash"`
	Preimage                 string            `json:"preimage" bun:",nullzero"`
	Internal                 bool              `json:"-" bun:",nullzero"`
	Keysend                  bool              `json:"keysend" bun:",nullzero"`
	State                    string            `json:"state" bun:",default:'initialized'"`
	ErrorMessage             string            `json:"error_message,omitempty" bun:",nullzero"`
	AddIndex                 uint64            `json:"-" bun:",nullzero"`
	CreatedAt                time.Time         `json:"created_at" bun:",nullzero,notnull,default:current_timestamp"`
	ExpiresAt                bun.NullTime      `json:"expires_at" bun:",nullzero"`
	UpdatedAt                bun.NullTime      `json:"updated_at"`
	SettledAt                bun.NullTime      `json:"settled_at"`
}

func (i *Invoice) SetFee(txEntry TransactionEntry, routingFee int64) {
	i.RoutingFee = routingFee
	i.Fee = routingFee
	if txEntry.ServiceFee != nil {
		i.ServiceFee = txEntry.ServiceFee.Amount
		i.Fee += txEntry.ServiceFee.Amount
	}
}

func (i *Invoice) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		i.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*Invoice)(nil)

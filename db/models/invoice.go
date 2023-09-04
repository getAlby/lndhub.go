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
	Amount                   int64             `json:"amount" validate:"gte=0"`
	Fee                      int64             `json:"fee" bun:",nullzero"`
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

type WebhookInvoicePayload struct {
	ID                       int64             `json:"id"`
	Type                     string            `json:"type"`
	UserLogin                string            `json:"user_login"`
	Amount                   int64             `json:"amount"`
	Fee                      int64             `json:"fee"`
	Balance                  int64             `json:"balance"`
	Memo                     string            `json:"memo"`
	DescriptionHash          string            `json:"description_hash,omitempty"`
	PaymentRequest           string            `json:"payment_request"`
	DestinationPubkeyHex     string            `json:"destination_pubkey_hex"`
	DestinationCustomRecords map[uint64][]byte `json:"custom_records,omitempty"`
	RHash                    string            `json:"r_hash"`
	Preimage                 string            `json:"preimage"`
	Keysend                  bool              `json:"keysend"`
	State                    string            `json:"state"`
	ErrorMessage             string            `json:"error_message,omitempty"`
	CreatedAt                time.Time         `json:"created_at"`
	ExpiresAt                time.Time         `json:"expires_at"`
	UpdatedAt                time.Time         `json:"updated_at"`
	SettledAt                time.Time         `json:"settled_at"`
}

func (i *Invoice) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		i.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*Invoice)(nil)

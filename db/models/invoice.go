package models

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/uptrace/bun"
)

// Invoice : Invoice Model
type Invoice struct {
	ID                   int64        `json:"id" bun:",pk,autoincrement"`
	Type                 string       `json:"type" validate:"required"`
	UserID               int64        `json:"user_id" validate:"required"`
	User                 *User        `bun:"rel:belongs-to,join:user_id=id"`
	Amount               int64        `json:"amount" validate:"gte=0"`
	Memo                 string       `json:"memo" bun:",nullzero"`
	DescriptionHash      string       `json:"description_hash" bun:",nullzero"`
	PaymentRequest       string       `json:"payment_request" bun:",nullzero"`
	DestinationPubkeyHex string       `json:"destination_pubkey_hex" bun:",notnull"`
	RHash                string       `json:"r_hash"`
	Preimage             string       `json:"preimage" bun:",nullzero"`
	Internal             bool         `json:"internal" bun:",nullzero"`
	State                string       `json:"state" bun:",default:'initialized'"`
	AddIndex             uint64       `json:"add_index" bun:",nullzero"`
	CreatedAt            time.Time    `bun:",nullzero,notnull,default:current_timestamp"`
	ExpiresAt            bun.NullTime `bun:",nullzero"`
	UpdatedAt            bun.NullTime `json:"updated_at"`
	SettledAt            bun.NullTime `json:"settled_at"`
}

func (i *Invoice) DestinationPubkey() (*btcec.PublicKey, error) {
	hexPubkey, err := hex.DecodeString(i.DestinationPubkeyHex)
	if err != nil {
		return nil, err
	}
	pubkey, err := btcec.ParsePubKey(hexPubkey[:], btcec.S256())
	if err != nil {
		return nil, err
	}
	return pubkey, nil
}

func (i *Invoice) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		i.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*Invoice)(nil)

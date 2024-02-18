package models

import (
	"time"
	"context"
	"github.com/uptrace/bun"
)
// Relay : Relay Model
type Relay struct {
	ID         int64 `bun:",pk,autoincrement"`
	Uri		   string `bun:",notnull,unique"`
	RelayName  string `bun:",notnull"`
	CreatedAt  time.Time     `bun:",notnull,default:current_timestamp"`
	UpdatedAt  bun.NullTime  `bun:",nullzero"`
	// relationship
	Filter	   *Filter `bun:"rel:has-one,join:id=relay_id"`
}
// * NOTE this is used so that the ORM applies the updated_at field
func (r *Relay) BeforeAppendModel(ctx context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		r.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

var _ bun.BeforeAppendModelHook = (*Relay)(nil)


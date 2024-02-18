package models

import (
	"time"
	"github.com/uptrace/bun"
)
// Filter : Filter Model
type Filter struct {
	RelayID   int64 `bun:",pk"`
	LastEventSeen int64 `bun:",nullzero"`
	CreatedAt time.Time `bun:",notnull,default:current_timestamp"`
	UpdatedAt bun.NullTime `bun:",nullzero"`
}
// * NOTE the filter model has a BeforeAppendModel hook too.
func (f *Filter) BeforeAppendModel(query bun.Query) error {
	switch query.(type) {
	case *bun.UpdateQuery:
		f.UpdatedAt = bun.NullTime{Time: time.Now()}
	}
	return nil
}

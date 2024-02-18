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


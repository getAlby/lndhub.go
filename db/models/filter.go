package models
import (
	"time"
	"github.com/uptrace/bun"
)
// Filter : Filter Model
type Filter struct {
	RelayId   string `bun:",pk"`
	LastEventSeen int64 `bun:",nullzero"`
	CreatedAt time.Time `bun:",notnull,default:current_timestamp"`
	UpdatedAt bun.NullTime `bun:",nullzero"`
	// relationship
	Relay *Relay `bun:"rel:has-one,join:relay_id"`
}


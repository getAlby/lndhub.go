package models

import (
	"time"
	"github.com/uptrace/bun"
)

// TODO - Determine if there should be any other fields from one of these RPCs
// * https://lightning.engineering/api-docs/api/taproot-assets/universe/query-asset-stats#universerpcassetstatsasset
// *

// Asset : Asset Model

// `AssetID` is not unique ! This is because `Group`s of the same
// Taproot Asset have different asset_id's.
type Asset struct {
	ID        int64     `bun:",pk,autoincrement"`
	AssetID   string    `bun:",notnull"`
	AssetName string    `bun:",notnull"`
	AssetType string    `bun:",notnull"`
	CreatedAt time.Time `bun:",nullzero,notnull,default:current_timestamp"`
	UpdatedAt bun.NullTime 
}


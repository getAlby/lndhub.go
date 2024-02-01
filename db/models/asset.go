package models;
// Asset : Asset model

// TODO examine what else we may want to track from these Tapd RPCs
// * https://lightning.engineering/api-docs/api/taproot-assets/taproot-assets/list-assets#taprpclistassetresponse
// * https://lightning.engineering/api-docs/api/taproot-assets/universe/query-asset-stats
type Asset struct {
	ID                 int64 `bun:",pk,autoincrement"`
	Name               string `bun:"unique,notnull"` // i.e. first is bitcoin
	DisplayGranularity int64

}
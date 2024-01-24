package models

// Account : Account Model
type Account struct {
	ID      int64  `bun:",pk,autoincrement"`
	UserID  int64  `bun:",notnull"`
	User    *User  `bun:"rel:belongs-to,join:user_id=id"`
	AssetID int64  `bun:",notnull"`
	Asset   *Asset `bun:"rel:has-one,join:asset_id=id"`
	Type    string `bun:",notnull"`
}

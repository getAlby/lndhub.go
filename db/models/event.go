package models
// Event : Event Model
type Event struct {
	ID         string `bun:",pk,autoincrement"`
	EventID    string `bun:",notnull,unique"`
	FromPubkey string `bun:",notnull"`
	Kind       int64  `bun:",notnull"`
	Content    string `bun:",notnull"`
	CreatedAt  int64  `bun:",notnull"`
}


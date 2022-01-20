package migrations

import (
	"embed"
	"log"

	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

//go:embed *.sql
var sqlMigrations embed.FS

func init() {
	if err := Migrations.Discover(sqlMigrations); err != nil {
		log.Fatalf("Error discovering migrations: %v", err)
	}
}

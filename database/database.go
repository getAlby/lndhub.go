package database

import (
	"github.com/bumi/lndhub.go/database/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	sqlite3    = "sqlite3"
	postgresql = "postgres"
)

//func configure() *gorm.Config {
//
//}

// Connect : Database connect
func Connect(database string) *gorm.DB {
	if database == sqlite3 {
		db, err := gorm.Open(sqlite.Open("./database/data.db"), &gorm.Config{})

		if err != nil {
			panic(err)
		}

		models.Migrate(db)

		return db
	} else if database == postgresql {
		dsn := "user=gorm1 password=gorm1 dbname=gorm1 port=5432"
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

		if err != nil {
			panic(err)
		}

		models.Migrate(db)

		return db
	} else {
		return nil
	}
}

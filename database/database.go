package database

import (
	"github.com/bumi/lndhub.go/database/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Connect : Database connect
func Connect(database string) *gorm.DB {
	db, err := gorm.Open(database, "user=gorm1 password=gorm1 dbname=gorm1 port=5432")
	//db, err := gorm.Open(database, "./database/data.db")
	db.LogMode(true)

	if err != nil {
		panic(err)
	}

	models.Migrate(db)

	return db
}

package database

import (
	"github.com/bumi/lndhub.go/database/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Connect : Database connect
func Connect(database string) *gorm.DB {
	//db, err := gorm.Open(database, "user=gorm password=gorm dbname=gorm")
	db, err := gorm.Open(database, "./database/data.db")
	db.LogMode(true)

	if err != nil {
		panic(err)
	}

	models.Migrate(db)

	return db
}

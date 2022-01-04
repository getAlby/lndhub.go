package database

import (
	"github.com/bumi/lndhub.go/database/models"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

// Connect : Database connect
func Connect(database string) *gorm.DB {
	//dsn := "host=localhost user=gorm password=gorm dbname=gorm port=9920 sslmode=disable"
	db, err := gorm.Open(database, "./database/data.db")
	db.LogMode(true)

	if err != nil {
		panic(err)
	}

	models.Migrate(db)

	return db
}

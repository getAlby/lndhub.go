package database

import (
	"errors"

	"github.com/bumi/lndhub.go/database/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	Sqlite3    = "sqlite3"
	Postgresql = "postgres"
)

func getDbDialect(database string) (*gorm.Dialector, error) {
	var dbOpen gorm.Dialector
	var err error
	if database == Sqlite3 {
		sqliteDbUri := "./database/data.db"
		dbOpen = sqlite.Open(sqliteDbUri)
	} else if database == Postgresql {
		postgresDbUri := "host=localhost user=gorm1 password=gorm1 dbname=gorm1 port=5432 sslmode=disable TimeZone=Asia/Shanghai"
		dbOpen = postgres.Open(postgresDbUri)
	} else {
		dbOpen = nil
		err = errors.New("non supported db dialect")
	}
	return &dbOpen, err
}

// Connect : Database connect
func Connect(database string) (*gorm.DB, error) {
	dbOpen, err := getDbDialect(database)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(*dbOpen, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = models.Migrate(db)
	if err != nil {
		return nil, err
	}

	return db, err
}

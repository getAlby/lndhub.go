package database

import (
	"github.com/bumi/lndhub.go/database/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"strings"
)

func getDbDialect(databaseURI string) (*gorm.Dialector, error) {
	var dbOpen gorm.Dialector
	var err error
	if strings.Contains(databaseURI, "postgresql") {
		postgresDbUri := databaseURI
		dbOpen = postgres.Open(postgresDbUri)
	} else {
		sqliteDbUri := databaseURI
		dbOpen = sqlite.Open(sqliteDbUri)
	}

	return &dbOpen, err
}

// Connect : Database connect
func Connect(databaseURI string) (*gorm.DB, error) {
	dbOpen, err := getDbDialect(databaseURI)
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

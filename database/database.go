package database

import (
	"errors"
	"strings"

	"github.com/bumi/lndhub.go/database/models"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func getDbDialect(databaseURI string) (*gorm.Dialector, error) {
	var dbOpen gorm.Dialector
	var err error
	if strings.HasPrefix(databaseURI, "postgresql:") {
		dbOpen = postgres.Open(databaseURI)
	} else if strings.HasPrefix(databaseURI, "sqlite:") {
		dbOpen = sqlite.Open(strings.Replace(databaseURI, "sqlite://", "", 1))
	} else {
		err = errors.New("invalid database configuration")
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

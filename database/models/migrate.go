package models

import "gorm.io/gorm"

// Migrate : migrate models
func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(&User{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&Account{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&Invoice{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&TransactionEntries{})
	if err != nil {
		return err
	}
	return nil
}

package models

import "gorm.io/gorm"

// Migrate : migrate models
func Migrate(db *gorm.DB) {
	db.AutoMigrate(&User{})
	db.AutoMigrate(&Account{})
	db.AutoMigrate(&Invoice{})
	db.AutoMigrate(&TransactionEntries{})
}

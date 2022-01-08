package models

import (
	"database/sql"
	"time"
)

// User : User Model
type User struct {
	ID           uint           `gorm:"primary_key"`
	Email        sql.NullString `gorm:uniqueIndex`
	Login        string         `gorm:"uniqueIndex;not null"`
	Password     string         `gorm:"index;not null"`
	RefreshToken sql.NullString `gorm:"uniqueIndex"`
	AccessToken  sql.NullString `gorm:"uniqueIndex"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
}

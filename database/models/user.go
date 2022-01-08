package models

import (
	"database/sql"
	"time"
)

// User : User Model
type User struct {
	ID           uint `gorm:"primary_key"`
	Email        sql.NullString
	Login        string         `gorm:"index"`
	Password     string         `gorm:"index"`
	RefreshToken sql.NullString `gorm:"index"`
	AccessToken  sql.NullString `gorm:"index"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
}

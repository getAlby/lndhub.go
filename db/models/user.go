package models

import (
	"database/sql"
	"time"

	"github.com/uptrace/bun"
)

// User : User Model
type User struct {
	bun.BaseModel `bun:"user"`

	ID        uint           `gorm:"primary_key"`
	Email     sql.NullString `gorm:"uniqueIndex"`
	Login     string         `gorm:"uniqueIndex;not null"`
	Password  string         `gorm:"index;not null"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
}

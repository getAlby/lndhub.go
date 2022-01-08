package models

import (
	"time"
)

// User : User Model
type User struct {
	ID           uint `gorm:"primary_key"`
	Email        string
	Login        string
	Password     string
	RefreshToken string
	AccessToken  string
	UpdatedAt    time.Time
	CreatedAt    time.Time
}

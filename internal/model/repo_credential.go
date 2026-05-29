package model

import "time"

type RepoCredential struct {
	ID        int64     `gorm:"primaryKey"`
	Name      string    `gorm:"not null"`
	Username  string    `gorm:"not null"`
	Password  string    `gorm:"not null"`
	CreatedBy int64     `gorm:"not null;default:0;index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

package model

import "time"

type SensitiveWord struct {
	ID          int64     `gorm:"primaryKey"`
	Original    string    `gorm:"not null"`
	Replacement string    `gorm:"not null"`
	CreatedAt   time.Time
}

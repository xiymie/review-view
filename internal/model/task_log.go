package model

import "time"

type TaskLogLevel string

const (
	TaskLogLevelInfo  TaskLogLevel = "info"
	TaskLogLevelWarn  TaskLogLevel = "warn"
	TaskLogLevelError TaskLogLevel = "error"
)

type TaskLog struct {
	ID        int64        `gorm:"primaryKey"`
	TaskID    int64        `gorm:"not null;index"`
	Level     TaskLogLevel `gorm:"not null"`
	Message   string       `gorm:"not null"`
	CreatedAt time.Time
}

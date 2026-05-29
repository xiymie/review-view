package model

import "time"

type OverflowStrategy string

const (
	OverflowStrategyQueue  OverflowStrategy = "queue"
	OverflowStrategyReject OverflowStrategy = "reject"
)

type ProjectStatus string

const (
	ProjectStatusInitializing ProjectStatus = "initializing"
	ProjectStatusReady        ProjectStatus = "ready"
	ProjectStatusInitFailed   ProjectStatus = "init_failed"
)

type Project struct {
	ID                 int64            `gorm:"primaryKey"`
	Name               string           `gorm:"not null"`
	RepoURL            string           `gorm:"not null"`
	Branch             string           `gorm:"not null"`
	ModelConfigID      int64            `gorm:"not null;index"`
	CustomPrompt       string
	LastReviewedCommit string
	OverflowStrategy   OverflowStrategy `gorm:"not null"`
	Status             ProjectStatus    `gorm:"not null;default:'initializing'"`
	InitError          string
	TaskTimeout        *int
	RepoCredentialID   *int64 `gorm:"index"`
	CronExpression     string
	CronEnabled        bool       `gorm:"not null;default:false"`
	NextRunAt          *time.Time
	CreatedBy          int64      `gorm:"not null;default:0;index"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

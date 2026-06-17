package model

import "time"

type TaskStatus string
type TaskTriggeredBy string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusRejected  TaskStatus = "rejected"
	TaskStatusSkipped   TaskStatus = "skipped"

	TaskTriggeredByManual  TaskTriggeredBy = "manual"
	TaskTriggeredByWebhook TaskTriggeredBy = "webhook"
	TaskTriggeredByCron    TaskTriggeredBy = "cron"
)

type Task struct {
	ID                  int64           `gorm:"primaryKey"`
	ProjectID           int64           `gorm:"not null;index"`
	Status              TaskStatus      `gorm:"not null;index"`
	FromCommit          string
	ToCommit            string          `gorm:"not null;index"`
	FromSubject         string // from_commit 的提交说明（区间起点；from 为空时为空）
	ToSubject           string // to_commit 的提交说明（区间终点）
	DiffContent         string
	CommitMessages      string // git log --oneline 格式的 commit 记录
	Result              string
	ErrorMessage        string
	TriggeredBy         TaskTriggeredBy `gorm:"not null"`
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	CreatedAt           time.Time
	StartedAt           *time.Time
	FinishedAt          *time.Time
}

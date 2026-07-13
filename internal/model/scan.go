package model

import "time"

// ScanSchedule 巡检配置，一条记录对应一个仓库的定时巡检任务
type ScanSchedule struct {
	ID               int64  `gorm:"primaryKey"`
	Name             string `gorm:"not null"`
	RepoURL          string `gorm:"not null"`
	RepoCredentialID *int64 `gorm:"index"`
	ModelConfigID    int64  `gorm:"not null;index"`
	// 巡检时间，格式 HH:MM，如 "09:00"；空则使用全局默认
	ScanTime string
	// 项目自定义提示词；空则使用全局默认
	CustomPrompt string
	// NAS WebDAV 地址，如 http://192.168.1.100:5005；空则使用全局默认
	NasURL      string
	NasUsername string
	NasPassword string
	// 上传子目录，默认用仓库名（文件路径格式：/TMMTMM/代码巡检记录/{SubDir}/...）
	NasSubDir string
	Enabled   bool      `gorm:"not null;default:true"`
	// BranchCheckpoints JSON map[branch]lastCommitHash，记录每个分支上次巡检到的 commit
	BranchCheckpoints string `gorm:"type:text"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ScanJobStatus 巡检执行状态
type ScanJobStatus string

const (
	ScanJobStatusPending   ScanJobStatus = "pending"
	ScanJobStatusRunning   ScanJobStatus = "running"
	ScanJobStatusCompleted ScanJobStatus = "completed"
	ScanJobStatusFailed    ScanJobStatus = "failed"
)

// ScanJob 每次巡检执行记录
type ScanJob struct {
	ID                 int64         `gorm:"primaryKey"`
	ScheduleID         int64         `gorm:"not null;index"`
	Status             ScanJobStatus `gorm:"not null;default:'pending'"`
	ErrorMessage       string
	BranchCount        int // 扫描的总分支数
	ChangedBranchCount int // 有改动的分支数
	ReportPath         string // 写入 NAS 的路径
	TriggeredAt        time.Time
	FinishedAt         *time.Time
}

// ScanAnalysisStage 分析阶段
type ScanAnalysisStage string

const (
	ScanAnalysisStageMessageOnly  ScanAnalysisStage = "message_only"  // 仅凭 commit message 判断
	ScanAnalysisStageWithDiff     ScanAnalysisStage = "with_diff"     // 追加了 diff 分析
)

// ScanBranchResult 每个分支的分析结果
type ScanBranchResult struct {
	ID             int64             `gorm:"primaryKey"`
	JobID          int64             `gorm:"not null;index"`
	BranchName     string            `gorm:"not null"`
	CommitCount    int
	CommitMessages string            // JSON 数组，存储 commit hash+message+author+time
	FromCommit     string
	ToCommit       string
	AnalysisStage  ScanAnalysisStage `gorm:"not null"`
	Result         string            // LLM 分析结果 Markdown
	HasRisk        bool              `gorm:"not null;default:false"`
	RiskLevel      string            // none / low / medium / high
	CreatedAt      time.Time
}

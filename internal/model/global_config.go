package model

import "time"

const (
	GlobalConfigKeyMaxConcurrentTasks = "max_concurrent_tasks"
	GlobalConfigKeyOverflowStrategy   = "global_overflow_strategy"
	GlobalConfigKeyRepoBaseDir        = "repo_base_dir"
	GlobalConfigKeyTaskTimeout        = "task_timeout"

	GlobalConfigKeySMTPHost     = "smtp_host"
	GlobalConfigKeySMTPPort     = "smtp_port"
	GlobalConfigKeySMTPUsername = "smtp_username"
	GlobalConfigKeySMTPPassword = "smtp_password"
	GlobalConfigKeySMTPFrom     = "smtp_from"
	GlobalConfigKeySMTPFromName = "smtp_from_name"
	GlobalConfigKeySMTPTLS      = "smtp_tls"

	GlobalConfigKeyScheduledScanUnchanged = "scheduled_scan_unchanged" // 定时扫描无新提交是否扫描，默认 false（跳过）
	GlobalConfigKeyManualScanUnchanged    = "manual_scan_unchanged"    // 手动扫描无新提交是否扫描，默认 true（扫描）
)

type GlobalConfig struct {
	Key       string `gorm:"primaryKey"`
	Value     string `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

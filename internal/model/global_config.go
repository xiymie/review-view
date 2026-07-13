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

	// 巡检全局配置
	GlobalConfigKeyScanTime        = "scan_default_time"    // 默认巡检时间 HH:MM，如 "09:00"
	GlobalConfigKeyScanPrompt      = "scan_default_prompt"  // 默认巡检提示词
	GlobalConfigKeyScanNasURL      = "scan_nas_url"         // NAS WebDAV 地址
	GlobalConfigKeyScanNasUsername = "scan_nas_username"    // NAS 用户名
	GlobalConfigKeyScanNasPassword = "scan_nas_password"    // NAS 密码
	GlobalConfigKeyScanRetainDays  = "scan_retain_days"     // NAS 报告保留天数，0=永久保留
)

type GlobalConfig struct {
	Key       string `gorm:"primaryKey"`
	Value     string `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

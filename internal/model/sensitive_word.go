package model

import "time"

// 敏感词类型
const (
	SensitiveWordTypeReplace = "replace" // 替换：发送给大模型前脱敏，收到响应后还原
	SensitiveWordTypeDetect  = "detect"  // 检测：扫描本地代码并将命中拼入审核报告
)

type SensitiveWord struct {
	ID          int64  `gorm:"primaryKey"`
	Type        string `gorm:"not null;default:replace"` // replace / detect
	Original    string `gorm:"not null"`
	Replacement string `gorm:"not null"`
	CreatedAt   time.Time
}

package model

import "time"

// ReviewSkill 表示可注入 Review Workflow 的审查技能配置。
type ReviewSkill struct {
	ID          int64  `gorm:"primaryKey"`
	Name        string `gorm:"not null;uniqueIndex"`
	Description string `gorm:"type:text"`
	Prompt      string `gorm:"type:text;not null"`
	Enabled     bool   `gorm:"not null;default:false;index"`
	BuiltIn     bool   `gorm:"not null;default:false"`
	SortOrder   int    `gorm:"not null;default:0;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	// Skill Registry 扩展字段：保留给未来官方 Skill Registry 结构使用，
	// 当前审查注入逻辑仅使用 Prompt 字段，这些字段默认为空。
	AgentXML          string `gorm:"type:text"`
	SkillRegistryXML  string `gorm:"type:text"`
	ToolRegistryXML   string `gorm:"type:text"`
	PolicyMD          string `gorm:"type:text"`
	WorkflowMD        string `gorm:"type:text"`
	ContextSchemaJSON string `gorm:"type:text"`
	MemorySchemaJSON  string `gorm:"type:text"`
	MetadataJSON      string `gorm:"type:text"`
}

package model

// ProjectReviewSkill 记录项目与 ReviewSkill 的多对多关联。
// 若项目无任何记录，则该项目不注入任何 Skill。
type ProjectReviewSkill struct {
	ProjectID     int64 `gorm:"primaryKey;not null;index"`
	ReviewSkillID int64 `gorm:"primaryKey;not null;index"`
}

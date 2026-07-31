package gormstore

import (
	"time"

	"review-view/internal/model"
	"gorm.io/gorm"
)

type ProjectStore struct {
	db *gorm.DB
}

func (s *ProjectStore) Create(project *model.Project) error {
	return s.db.Create(project).Error
}

func (s *ProjectStore) Update(project *model.Project) error {
	return s.db.Save(project).Error
}

func (s *ProjectStore) Delete(id int64) error {
	return s.db.Delete(&model.Project{}, id).Error
}

func (s *ProjectStore) GetByID(id int64) (*model.Project, error) {
	var project model.Project
	if err := s.db.First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *ProjectStore) List() ([]model.Project, error) {
	var projects []model.Project
	if err := s.db.Order("id asc").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *ProjectStore) ListByUser(userID int64) ([]model.Project, error) {
	var projects []model.Project
	if err := s.db.Where("created_by = ?", userID).Order("id asc").Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// RecoverInitializing 将所有 initializing 状态的项目标记为 init_failed，返回受影响的行数
func (s *ProjectStore) RecoverInitializing() (int64, error) {
	result := s.db.Model(&model.Project{}).
		Where("status = ?", model.ProjectStatusInitializing).
		Updates(map[string]interface{}{
			"status":     model.ProjectStatusInitFailed,
			"init_error": "服务重启，初始化被中断",
		})
	return result.RowsAffected, result.Error
}

func (s *ProjectStore) ListCronEnabled() ([]*model.Project, error) {
	var projects []*model.Project
	if err := s.db.Where("cron_enabled = ?", true).Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *ProjectStore) UpdateNextRunAt(id int64, t *time.Time) error {
	return s.db.Model(&model.Project{}).Where("id = ?", id).Update("next_run_at", t).Error
}

// SetSkills 替换项目关联的 ReviewSkill ID 列表，在同一事务内完成删旧插新。
func (s *ProjectStore) SetSkills(projectID int64, skillIDs []int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&model.ProjectReviewSkill{}).Error; err != nil {
			return err
		}
		if len(skillIDs) == 0 {
			return nil
		}
		rows := make([]model.ProjectReviewSkill, 0, len(skillIDs))
		for _, sid := range skillIDs {
			rows = append(rows, model.ProjectReviewSkill{ProjectID: projectID, ReviewSkillID: sid})
		}
		return tx.Create(&rows).Error
	})
}

// ListSkillIDs 返回项目已选的 ReviewSkill ID 列表。
func (s *ProjectStore) ListSkillIDs(projectID int64) ([]int64, error) {
	var rows []model.ProjectReviewSkill
	if err := s.db.Where("project_id = ?", projectID).Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ReviewSkillID)
	}
	return ids, nil
}

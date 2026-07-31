package gormstore

import (
	"review-view/internal/model"

	"gorm.io/gorm"
)

type ReviewSkillStore struct {
	db *gorm.DB
}

func (s *ReviewSkillStore) Create(skill *model.ReviewSkill) error {
	return s.db.Select("*").Create(skill).Error
}

func (s *ReviewSkillStore) Update(skill *model.ReviewSkill) error {
	return s.db.Save(skill).Error
}

func (s *ReviewSkillStore) Delete(id int64) error {
	return s.db.Delete(&model.ReviewSkill{}, id).Error
}

func (s *ReviewSkillStore) GetByID(id int64) (*model.ReviewSkill, error) {
	var skill model.ReviewSkill
	if err := s.db.First(&skill, id).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

func (s *ReviewSkillStore) GetByName(name string) (*model.ReviewSkill, error) {
	var skill model.ReviewSkill
	if err := s.db.First(&skill, "name = ?", name).Error; err != nil {
		return nil, err
	}
	return &skill, nil
}

func (s *ReviewSkillStore) List() ([]model.ReviewSkill, error) {
	var skills []model.ReviewSkill
	if err := s.db.Order("sort_order asc, id asc").Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

func (s *ReviewSkillStore) ListEnabled() ([]model.ReviewSkill, error) {
	var skills []model.ReviewSkill
	if err := s.db.Where("enabled = ?", true).Order("sort_order asc, id asc").Find(&skills).Error; err != nil {
		return nil, err
	}
	return skills, nil
}

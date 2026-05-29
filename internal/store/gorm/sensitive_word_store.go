package gormstore

import (
	"review-view/internal/model"

	"gorm.io/gorm"
)

type SensitiveWordStore struct {
	db *gorm.DB
}

func (s *SensitiveWordStore) Create(w *model.SensitiveWord) error {
	return s.db.Create(w).Error
}

func (s *SensitiveWordStore) Update(w *model.SensitiveWord) error {
	return s.db.Save(w).Error
}

func (s *SensitiveWordStore) Delete(id int64) error {
	return s.db.Delete(&model.SensitiveWord{}, id).Error
}

func (s *SensitiveWordStore) List() ([]model.SensitiveWord, error) {
	var words []model.SensitiveWord
	return words, s.db.Order("id asc").Find(&words).Error
}

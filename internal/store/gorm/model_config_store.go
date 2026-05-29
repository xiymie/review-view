package gormstore

import (
	"review-view/internal/model"
	"gorm.io/gorm"
)

type ModelConfigStore struct {
	db *gorm.DB
}

func (s *ModelConfigStore) Create(config *model.ModelConfig) error {
	return s.db.Create(config).Error
}

func (s *ModelConfigStore) Update(config *model.ModelConfig) error {
	return s.db.Save(config).Error
}

func (s *ModelConfigStore) GetByID(id int64) (*model.ModelConfig, error) {
	var config model.ModelConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (s *ModelConfigStore) Delete(id int64) error {
	return s.db.Delete(&model.ModelConfig{}, id).Error
}

func (s *ModelConfigStore) CountByModelConfig(modelConfigID int64) (int64, error) {
	var count int64
	err := s.db.Model(&model.Project{}).Where("model_config_id = ?", modelConfigID).Count(&count).Error
	return count, err
}

func (s *ModelConfigStore) List() ([]model.ModelConfig, error) {
	var configs []model.ModelConfig
	if err := s.db.Order("id asc").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

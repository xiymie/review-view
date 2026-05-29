package gormstore

import (
	"errors"

	"review-view/internal/model"
	"gorm.io/gorm"
)

type GlobalConfigStore struct {
	db *gorm.DB
}

var defaultGlobalConfigs = map[string]string{
	model.GlobalConfigKeyMaxConcurrentTasks: "3",
	model.GlobalConfigKeyOverflowStrategy:   string(model.OverflowStrategyQueue),
	model.GlobalConfigKeyRepoBaseDir:        "./repos",
	model.GlobalConfigKeyTaskTimeout:        "30",
}

func (s *GlobalConfigStore) EnsureDefaults() error {
	for key, value := range defaultGlobalConfigs {
		err := s.db.FirstOrCreate(&model.GlobalConfig{Key: key}, model.GlobalConfig{Key: key, Value: value}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *GlobalConfigStore) Get(key string) (string, error) {
	var config model.GlobalConfig
	if err := s.db.First(&config, "key = ?", key).Error; err != nil {
		return "", err
	}
	return config.Value, nil
}

func (s *GlobalConfigStore) Set(key, value string) error {
	return s.db.Save(&model.GlobalConfig{Key: key, Value: value}).Error
}

func (s *GlobalConfigStore) List() ([]model.GlobalConfig, error) {
	var configs []model.GlobalConfig
	if err := s.db.Order("key asc").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *GlobalConfigStore) GetOrNil(key string) (*model.GlobalConfig, error) {
	var config model.GlobalConfig
	err := s.db.First(&config, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

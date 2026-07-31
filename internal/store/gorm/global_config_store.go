package gormstore

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"review-view/internal/model"
)

type GlobalConfigStore struct {
	db *gorm.DB
}

var defaultGlobalConfigs = map[string]string{
	model.GlobalConfigKeyMaxConcurrentTasks: "3",
	model.GlobalConfigKeyOverflowStrategy:   string(model.OverflowStrategyQueue),
	model.GlobalConfigKeyRepoBaseDir:        "./repos",
	model.GlobalConfigKeyTaskTimeout:        "30",
	model.GlobalConfigKeyScanPrompt:         model.DefaultScanPrompt,
}

func (s *GlobalConfigStore) EnsureDefaults() error {
	for key, value := range defaultGlobalConfigs {
		if key == model.GlobalConfigKeyScanPrompt {
			if err := s.ensureScanPromptDefault(value); err != nil {
				return err
			}
			continue
		}
		if err := s.ensureDefault(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *GlobalConfigStore) ensureDefault(key, value string) error {
	var existing model.GlobalConfig
	err := s.db.First(&existing, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Set(key, value)
	}
	return err
}

func (s *GlobalConfigStore) ensureScanPromptDefault(value string) error {
	var existing model.GlobalConfig
	err := s.db.First(&existing, "key = ?", model.GlobalConfigKeyScanPrompt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.Set(model.GlobalConfigKeyScanPrompt, value)
	}
	if err != nil {
		return err
	}
	if existing.Value == "" || existing.Value == model.LegacyDefaultScanPrompt {
		return s.Set(model.GlobalConfigKeyScanPrompt, value)
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
	cfg := model.GlobalConfig{Key: key, Value: value}
	return s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&cfg).Error
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

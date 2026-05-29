package gormstore

import (
	"testing"

	"review-view/internal/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Stores struct {
	Projects       *ProjectStore
	ModelConfigs   *ModelConfigStore
	Tasks          *TaskStore
	GlobalConfigs  *GlobalConfigStore
	Credentials    *RepoCredentialStore
	SensitiveWords *SensitiveWordStore
	Users          *UserStore
}

func Open(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&model.Project{}, &model.ModelConfig{}, &model.Task{}, &model.TaskLog{}, &model.GlobalConfig{}, &model.RepoCredential{}, &model.SensitiveWord{}, &model.User{}); err != nil {
		return nil, err
	}
	return db, nil
}

func New(db *gorm.DB) Stores {
	return Stores{
		Projects:       &ProjectStore{db: db},
		ModelConfigs:   &ModelConfigStore{db: db},
		Tasks:          &TaskStore{db: db},
		GlobalConfigs:  &GlobalConfigStore{db: db},
		Credentials:    &RepoCredentialStore{db: db},
		SensitiveWords: &SensitiveWordStore{db: db},
		Users:          &UserStore{db: db},
	}
}

func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := Open("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

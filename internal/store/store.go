package store

import (
	"time"

	"review-view/internal/model"
)

type ProjectStore interface {
	Create(*model.Project) error
	Update(*model.Project) error
	Delete(id int64) error
	GetByID(id int64) (*model.Project, error)
	List() ([]model.Project, error)
	ListByUser(userID int64) ([]model.Project, error)
	RecoverInitializing() (int64, error)
	ListCronEnabled() ([]*model.Project, error)
	UpdateNextRunAt(id int64, t *time.Time) error
}

type ModelConfigStore interface {
	Create(*model.ModelConfig) error
	Update(*model.ModelConfig) error
	Delete(id int64) error
	GetByID(id int64) (*model.ModelConfig, error)
	List() ([]model.ModelConfig, error)
	CountByModelConfig(modelConfigID int64) (int64, error)
}

type TaskStore interface {
	Create(*model.Task) error
	Update(*model.Task) error
	Delete(id int64) error
	DeleteByProject(projectID int64) error
	GetByID(id int64) (*model.Task, error)
	ListPending(limit int) ([]model.Task, error)
	ListRecent(limit int) ([]model.Task, error)
	ListByProject(projectID int64, limit int) ([]model.Task, error)
	ListByProjectIDs(projectIDs []int64, limit int) ([]model.Task, error)
	FindCompletedRange(projectID int64, fromCommit, toCommit string) (*model.Task, error)
	FindActiveRange(projectID int64, fromCommit, toCommit string) (*model.Task, error)
	CountRunningByProject(projectID int64) (int64, error)
	ClaimPending(taskID int64) bool
	AppendLog(taskID int64, level model.TaskLogLevel, message string) error
	AppendLogs(logs []model.TaskLog) error
	ListLogs(taskID int64) ([]model.TaskLog, error)
	UpdateResult(taskID int64, result string) error
	// RecoverRunning 将所有 running 状态的任务标记为 failed，用于服务重启后清理残留状态
	RecoverRunning() (int64, error)
}

type GlobalConfigStore interface {
	EnsureDefaults() error
	Get(key string) (string, error)
	Set(key, value string) error
	List() ([]model.GlobalConfig, error)
}

type RepoCredentialStore interface {
	Create(*model.RepoCredential) error
	Update(*model.RepoCredential) error
	Delete(id int64) error
	GetByID(id int64) (*model.RepoCredential, error)
	List() ([]model.RepoCredential, error)
	ListByUser(userID int64) ([]model.RepoCredential, error)
	CountProjectsByCredential(credentialID int64) (int64, error)
}

type SensitiveWordStore interface {
	Create(*model.SensitiveWord) error
	Update(*model.SensitiveWord) error
	Delete(id int64) error
	List() ([]model.SensitiveWord, error)
}

type UserStore interface {
	Create(*model.User) error
	Update(*model.User) error
	Delete(id int64) error
	GetByID(id int64) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	List() ([]model.User, error)
	Count() (int64, error)
}

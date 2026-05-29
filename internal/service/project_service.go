package service

import (
	"context"
	"fmt"
	"log"
	"time"

	cronparser "github.com/robfig/cron/v3"
	"review-view/internal/model"
	"review-view/internal/review"
	"review-view/internal/store"
)

type ProjectCreateInput struct {
	Name              string
	RepoURL           string
	Branch            string
	ModelConfigID     int64
	CustomPrompt      string
	OverflowStrategy  model.OverflowStrategy
	TaskTimeout       *int
	RepoCredentialID  *int64
	CronExpression    string
	CronEnabled       bool
	CreatedBy         int64
}

type ProjectService struct {
	projects     store.ProjectStore
	modelConfigs store.ModelConfigStore
	tasks        store.TaskStore
	repoManager  *review.RepositoryManager
	credentials  store.RepoCredentialStore
}

func NewProjectService(projects store.ProjectStore, modelConfigs store.ModelConfigStore, tasks store.TaskStore, repoManager *review.RepositoryManager, credentials store.RepoCredentialStore) *ProjectService {
	return &ProjectService{
		projects:     projects,
		modelConfigs: modelConfigs,
		tasks:        tasks,
		repoManager:  repoManager,
		credentials:  credentials,
	}
}

func (s *ProjectService) List() ([]model.Project, error) {
	return s.projects.List()
}

func (s *ProjectService) ListByUser(userID int64) ([]model.Project, error) {
	return s.projects.ListByUser(userID)
}

func (s *ProjectService) Get(id int64) (*model.Project, error) {
	return s.projects.GetByID(id)
}

func (s *ProjectService) Create(input ProjectCreateInput) (*model.Project, error) {
	if _, err := s.modelConfigs.GetByID(input.ModelConfigID); err != nil {
		return nil, err
	}

	nextRunAt, err := computeNextRunAt(input.CronExpression, input.CronEnabled)
	if err != nil {
		return nil, err
	}

	project := &model.Project{
		Name:              input.Name,
		RepoURL:           input.RepoURL,
		Branch:            input.Branch,
		ModelConfigID:     input.ModelConfigID,
		CustomPrompt:      input.CustomPrompt,
		OverflowStrategy:  input.OverflowStrategy,
		TaskTimeout:       input.TaskTimeout,
		RepoCredentialID:  input.RepoCredentialID,
		CronExpression:    input.CronExpression,
		CronEnabled:       input.CronEnabled,
		NextRunAt:         nextRunAt,
		Status:            model.ProjectStatusInitializing,
		CreatedBy:         input.CreatedBy,
	}
	if err := s.projects.Create(project); err != nil {
		return nil, err
	}

	go s.initProject(project.ID, project.RepoURL, project.Branch)

	return project, nil
}

func (s *ProjectService) Update(id int64, input ProjectCreateInput) (*model.Project, error) {
	if _, err := s.modelConfigs.GetByID(input.ModelConfigID); err != nil {
		return nil, err
	}

	project, err := s.projects.GetByID(id)
	if err != nil {
		return nil, err
	}

	nextRunAt, err := computeNextRunAt(input.CronExpression, input.CronEnabled)
	if err != nil {
		return nil, err
	}

	project.Name = input.Name
	project.RepoURL = input.RepoURL
	project.Branch = input.Branch
	project.ModelConfigID = input.ModelConfigID
	project.CustomPrompt = input.CustomPrompt
	project.OverflowStrategy = input.OverflowStrategy
	project.TaskTimeout = input.TaskTimeout
	project.RepoCredentialID = input.RepoCredentialID
	project.CronExpression = input.CronExpression
	project.CronEnabled = input.CronEnabled
	project.NextRunAt = nextRunAt

	if err := s.projects.Update(project); err != nil {
		return nil, err
	}
	return project, nil
}

// Delete 删除项目及其所有关联任务和本地仓库，不允许删除有运行中任务的项目
func (s *ProjectService) Delete(id int64) error {
	running, err := s.tasks.CountRunningByProject(id)
	if err != nil {
		return err
	}
	if running > 0 {
		return fmt.Errorf("项目有 %d 个运行中的任务，无法删除", running)
	}

	if err := s.tasks.DeleteByProject(id); err != nil {
		return err
	}
	if err := s.projects.Delete(id); err != nil {
		return err
	}
	// 数据库记录删除成功后，清理本地仓库目录
	_ = s.repoManager.RemoveRepo(id)
	return nil
}

// initProject 异步执行项目初始化（clone + resolve HEAD）
func (s *ProjectService) initProject(projectID int64, repoURL, branch string) {
	ctx := context.Background()

	project, err := s.projects.GetByID(projectID)
	if err != nil {
		return
	}

	var cred *model.RepoCredential
	if project.RepoCredentialID != nil {
		cred, _ = s.credentials.GetByID(*project.RepoCredentialID)
	}

	repoDir, err := s.repoManager.EnsureRepo(ctx, projectID, repoURL, branch, cred)
	if err != nil {
		project.Status = model.ProjectStatusInitFailed
		project.InitError = err.Error()
		_ = s.projects.Update(project)
		log.Printf("project %d init failed: %v", projectID, err)
		return
	}

	_, err = s.repoManager.ResolveHeadCommit(ctx, repoDir, branch)
	if err != nil {
		project.Status = model.ProjectStatusInitFailed
		project.InitError = err.Error()
		_ = s.projects.Update(project)
		log.Printf("project %d resolve head failed: %v", projectID, err)
		return
	}

	project.Status = model.ProjectStatusReady
	project.InitError = ""
	_ = s.projects.Update(project)
	log.Printf("project %d initialized successfully", projectID)
}

// Initialize 重新初始化项目（用于 init_failed 状态的重试）
func (s *ProjectService) Initialize(id int64) error {
	project, err := s.projects.GetByID(id)
	if err != nil {
		return err
	}
	if project.Status != model.ProjectStatusInitFailed {
		return fmt.Errorf("project %d status is %s, not init_failed", id, project.Status)
	}

	project.Status = model.ProjectStatusInitializing
	project.InitError = ""
	if err := s.projects.Update(project); err != nil {
		return err
	}

	go s.initProject(project.ID, project.RepoURL, project.Branch)
	return nil
}

// computeNextRunAt 根据 cron 表达式计算下次执行时间；禁用或表达式为空时返回 nil
func computeNextRunAt(expr string, enabled bool) (*time.Time, error) {
	if !enabled || expr == "" {
		return nil, nil
	}
	sched, err := cronparser.ParseStandard(expr)
	if err != nil {
		return nil, fmt.Errorf("无效的 cron 表达式: %w", err)
	}
	t := sched.Next(time.Now())
	return &t, nil
}

// UpdateSchedule 仅更新项目的 cron 配置字段
func (s *ProjectService) UpdateSchedule(id int64, cronExpr string, cronEnabled bool) (*model.Project, error) {
	project, err := s.projects.GetByID(id)
	if err != nil {
		return nil, err
	}

	nextRunAt, err := computeNextRunAt(cronExpr, cronEnabled)
	if err != nil {
		return nil, err
	}

	project.CronExpression = cronExpr
	project.CronEnabled = cronEnabled
	project.NextRunAt = nextRunAt

	if err := s.projects.Update(project); err != nil {
		return nil, err
	}
	return project, nil
}

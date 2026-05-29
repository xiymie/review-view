package service

import (
	"context"
	"fmt"
	"time"

	"review-view/internal/model"
	"review-view/internal/review"
	"review-view/internal/store"
)

type TriggerInput struct {
	ProjectID    int64
	TriggeredBy  model.TaskTriggeredBy
	TargetCommit string
	FromCommit   string // 手动指定起始 commit（为空则使用 LastReviewedCommit）
}

type TaskService struct {
	projects     store.ProjectStore
	modelConfigs store.ModelConfigStore
	tasks        store.TaskStore
	repoManager  *review.RepositoryManager
	credentials  store.RepoCredentialStore
	settings     *SettingsService
}

func NewTaskService(projects store.ProjectStore, modelConfigs store.ModelConfigStore, tasks store.TaskStore, repoManager *review.RepositoryManager, credentials store.RepoCredentialStore, settings *SettingsService) *TaskService {
	return &TaskService{
		projects:     projects,
		modelConfigs: modelConfigs,
		tasks:        tasks,
		repoManager:  repoManager,
		credentials:  credentials,
		settings:     settings,
	}
}

// credForProject 根据项目的 RepoCredentialID 查询凭据，无凭据时返回 nil。
func (s *TaskService) credForProject(project *model.Project) *model.RepoCredential {
	if project.RepoCredentialID == nil {
		return nil
	}
	cred, _ := s.credentials.GetByID(*project.RepoCredentialID)
	return cred
}

func (s *TaskService) Trigger(ctx context.Context, input TriggerInput) (*model.Task, bool, error) {
	project, err := s.projects.GetByID(input.ProjectID)
	if err != nil {
		return nil, false, err
	}

	// 项目状态检查：非 ready 状态不允许触发
	if project.Status != model.ProjectStatusReady {
		return nil, false, fmt.Errorf("project status is %s, cannot trigger review", project.Status)
	}

	repoDir, err := s.repoManager.EnsureRepo(ctx, project.ID, project.RepoURL, project.Branch, s.credForProject(project))
	if err != nil {
		return nil, false, err
	}

	toCommit := input.TargetCommit
	if toCommit == "" {
		toCommit, err = s.repoManager.ResolveHeadCommit(ctx, repoDir, project.Branch)
		if err != nil {
			return nil, false, err
		}
	}

	// fromCommit 优先使用 TriggerInput 指定值，否则使用 project 的 LastReviewedCommit
	fromCommit := input.FromCommit
	if fromCommit == "" {
		fromCommit = project.LastReviewedCommit
	}
	if fromCommit == toCommit {
		if s.shouldSkipUnchanged(input.TriggeredBy) {
			finishedAt := time.Now()
			skippedTask := &model.Task{
				ProjectID:    project.ID,
				Status:       model.TaskStatusSkipped,
				FromCommit:   fromCommit,
				ToCommit:     toCommit,
				TriggeredBy:  input.TriggeredBy,
				ErrorMessage: "代码无新提交，跳过本次扫描",
				FinishedAt:   &finishedAt,
			}
			if err := s.tasks.Create(skippedTask); err != nil {
				return nil, false, err
			}
			return skippedTask, true, nil
		}
		return nil, true, nil
	}

	// 检查是否已有相同 commit 范围的活跃任务（pending/running），防止重复触发
	active, err := s.tasks.FindActiveRange(project.ID, fromCommit, toCommit)
	if err != nil {
		return nil, false, err
	}
	if active != nil {
		return nil, true, nil
	}

	existing, err := s.tasks.FindCompletedRange(project.ID, fromCommit, toCommit)
	if err != nil {
		return nil, false, err
	}
	if existing != nil {
		return nil, true, nil
	}

	task := &model.Task{
		ProjectID:   project.ID,
		Status:      model.TaskStatusPending,
		FromCommit:  fromCommit,
		ToCommit:    toCommit,
		TriggeredBy: input.TriggeredBy,
	}
	if err := s.populateTaskChanges(ctx, repoDir, task); err != nil {
		return nil, false, err
	}
	if err := s.tasks.Create(task); err != nil {
		return nil, false, err
	}

	runningCount, err := s.tasks.CountRunningByProject(project.ID)
	if err != nil {
		return nil, false, err
	}
	if runningCount > 0 && project.OverflowStrategy == model.OverflowStrategyReject {
		task.Status = model.TaskStatusRejected
		task.ErrorMessage = "project already has running task"
		if err := s.tasks.Update(task); err != nil {
			return nil, false, err
		}
	}

	return task, false, nil
}

func (s *TaskService) ListByProject(projectID int64, limit int) ([]model.Task, error) {
	return s.tasks.ListByProject(projectID, limit)
}

func (s *TaskService) Retry(ctx context.Context, taskID int64) (*model.Task, error) {
	task, err := s.tasks.GetByID(taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != model.TaskStatusFailed && task.Status != model.TaskStatusCancelled {
		return nil, fmt.Errorf("task %d cannot be retried from status %s", task.ID, task.Status)
	}

	project, err := s.projects.GetByID(task.ProjectID)
	if err != nil {
		return nil, err
	}
	repoDir, err := s.repoManager.EnsureRepo(ctx, project.ID, project.RepoURL, project.Branch, s.credForProject(project))
	if err != nil {
		return nil, err
	}

	retryTask := &model.Task{
		ProjectID:   task.ProjectID,
		Status:      model.TaskStatusPending,
		FromCommit:  task.FromCommit,
		ToCommit:    task.ToCommit,
		TriggeredBy: model.TaskTriggeredByManual,
	}
	if err := s.populateTaskChanges(ctx, repoDir, retryTask); err != nil {
		return nil, err
	}
	if err := s.tasks.Create(retryTask); err != nil {
		return nil, err
	}
	return retryTask, nil
}

// ListCommits 获取项目最近 N 条 commit 记录
func (s *TaskService) ListCommits(ctx context.Context, projectID int64, branch string, limit int) ([]review.CommitInfo, error) {
	project, err := s.projects.GetByID(projectID)
	if err != nil {
		return nil, err
	}
	repoDir, err := s.repoManager.EnsureRepo(ctx, project.ID, project.RepoURL, project.Branch, s.credForProject(project))
	if err != nil {
		return nil, err
	}
	return s.repoManager.ListCommits(ctx, repoDir, branch, limit)
}

// populateTaskChanges 获取变更文件列表和 commit 记录填充到 task 中。
// 由 Trigger 和 Retry 共用，避免重复代码。
// shouldSkipUnchanged 根据触发方式和全局配置，判断无新提交时是否应跳过
// cron 触发：默认跳过（scheduled_scan_unchanged=false）
// manual/webhook 触发：默认不跳过（manual_scan_unchanged=true）
func (s *TaskService) shouldSkipUnchanged(triggeredBy model.TaskTriggeredBy) bool {
	if s.settings == nil {
		// 无 settings 注入时走默认逻辑
		return triggeredBy == model.TaskTriggeredByCron
	}
	cfg, err := s.settings.Get()
	if err != nil {
		return triggeredBy == model.TaskTriggeredByCron
	}
	if triggeredBy == model.TaskTriggeredByCron {
		return !cfg.ScheduledScanUnchanged
	}
	// manual / webhook
	return !cfg.ManualScanUnchanged
}

func (s *TaskService) populateTaskChanges(ctx context.Context, repoDir string, task *model.Task) error {
	diffNameStatus, err := s.repoManager.BuildDiffNameStatus(ctx, repoDir, task.FromCommit, task.ToCommit)
	if err != nil {
		return fmt.Errorf("获取变更文件列表失败: %w", err)
	}
	if diffNameStatus == "" {
		return fmt.Errorf("commit 范围 %s..%s 没有变更文件", task.FromCommit, task.ToCommit)
	}
	commitLog, _ := s.repoManager.BuildCommitLog(ctx, repoDir, task.FromCommit, task.ToCommit)
	task.DiffContent = diffNameStatus
	task.CommitMessages = commitLog
	return nil
}

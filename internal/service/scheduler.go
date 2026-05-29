package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cronparser "github.com/robfig/cron/v3"
	"review-view/internal/model"
	"review-view/internal/notify"
	"review-view/internal/review"
	"review-view/internal/store"
	"golang.org/x/sync/semaphore"
)

type Scheduler struct {
	projects        store.ProjectStore
	modelConfigs    store.ModelConfigStore
	tasks           store.TaskStore
	globalConfigs   store.GlobalConfigStore
	repoManager     *review.RepositoryManager
	credentials     store.RepoCredentialStore
	users           store.UserStore
	reviewerFactory func(*model.ModelConfig) review.Reviewer
	sensitiveWords  *SensitiveWordService
	cache           *TaskCache
	taskService     *TaskService
	notifier        notify.Notifier
	sem             *semaphore.Weighted
	cancels         sync.Map
	interval        time.Duration
	execute         func(context.Context, int64) error
	onTaskLaunched  func(int64)
}

func NewScheduler(
	projects store.ProjectStore,
	modelConfigs store.ModelConfigStore,
	tasks store.TaskStore,
	globalConfigs store.GlobalConfigStore,
	repoManager *review.RepositoryManager,
	credentials store.RepoCredentialStore,
	reviewerFactory func(*model.ModelConfig) review.Reviewer,
	cache *TaskCache,
	maxConcurrent int64,
	interval time.Duration,
	sensitiveWords ...*SensitiveWordService,
) *Scheduler {
	if reviewerFactory == nil {
		reviewerFactory = review.GetReviewer
	}
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	var sw *SensitiveWordService
	if len(sensitiveWords) > 0 {
		sw = sensitiveWords[0]
	}

	scheduler := &Scheduler{
		projects:        projects,
		modelConfigs:    modelConfigs,
		tasks:           tasks,
		globalConfigs:   globalConfigs,
		repoManager:     repoManager,
		credentials:     credentials,
		reviewerFactory: reviewerFactory,
		sensitiveWords:  sw,
		cache:           cache,
		sem:             semaphore.NewWeighted(maxConcurrent),
		interval:        interval,
	}
	scheduler.execute = scheduler.ExecuteTask
	return scheduler
}

func (s *Scheduler) SetTaskService(ts *TaskService) {
	s.taskService = ts
}

func (s *Scheduler) SetNotifier(n notify.Notifier, users store.UserStore) {
	s.notifier = n
	s.users = users
}

func (s *Scheduler) RunOnce(ctx context.Context) error {
	pending, err := s.tasks.ListPending(20)
	if err != nil {
		return err
	}

	for i := range pending {
		task := &pending[i]

		// 原子抢占：将 pending 改为 running，防止被其他轮询重复拾取
		if !s.tasks.ClaimPending(task.ID) {
			continue
		}

		if err := s.sem.Acquire(ctx, 1); err != nil {
			return err
		}

		taskID := task.ID
		if s.onTaskLaunched != nil {
			s.onTaskLaunched(taskID)
		}

		go func() {
			defer s.sem.Release(1)
			_ = s.execute(context.Background(), taskID)
		}()
	}

	return nil
}

func (s *Scheduler) Loop(ctx context.Context) {
	if s.interval <= 0 {
		s.interval = 5 * time.Second
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	if s.taskService != nil {
		go s.cronLoop(ctx)
	}

	for {
		_ = s.RunOnce(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// cronLoop 每 30 秒扫描一次 cron_enabled 的项目，到期则触发任务
func (s *Scheduler) cronLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runCronScan(ctx)
		}
	}
}

func (s *Scheduler) runCronScan(ctx context.Context) {
	projects, err := s.projects.ListCronEnabled()
	if err != nil {
		log.Printf("cron scan: list projects: %v", err)
		return
	}

	now := time.Now()
	for _, p := range projects {
		if p.NextRunAt == nil || p.NextRunAt.After(now) {
			continue
		}

		_, _, err := s.taskService.Trigger(ctx, TriggerInput{
			ProjectID:   p.ID,
			TriggeredBy: model.TaskTriggeredByCron,
		})
		if err != nil {
			log.Printf("cron scan: trigger project %d: %v", p.ID, err)
		}

		nextRunAt, parseErr := cronNextTime(p.CronExpression)
		if parseErr != nil {
			log.Printf("cron scan: parse expression for project %d: %v", p.ID, parseErr)
			continue
		}
		if updateErr := s.projects.UpdateNextRunAt(p.ID, nextRunAt); updateErr != nil {
			log.Printf("cron scan: update next_run_at for project %d: %v", p.ID, updateErr)
		}
	}
}

// cronNextTime 解析 cron 表达式并返回下次执行时间
func cronNextTime(expr string) (*time.Time, error) {
	sched, err := cronparser.ParseStandard(expr)
	if err != nil {
		return nil, err
	}
	t := sched.Next(time.Now())
	return &t, nil
}

// appendLog 记录任务日志到内存 cache，不直接写 DB
func (s *Scheduler) appendLog(taskID int64, level model.TaskLogLevel, message string) {
	s.cache.AppendLog(taskID, level, message)
}

// flushLogs 将任务的缓冲日志刷盘到 DB，并清理内存
func (s *Scheduler) flushLogs(taskID int64) {
	s.cache.Flush(taskID)
}

func (s *Scheduler) ExecuteTask(ctx context.Context, taskID int64) error {
	task, err := s.tasks.GetByID(taskID)
	if err != nil {
		return err
	}

	project, err := s.projects.GetByID(task.ProjectID)
	if err != nil {
		return err
	}

	modelConfig, err := s.modelConfigs.GetByID(project.ModelConfigID)
	if err != nil {
		return err
	}

	timeoutMinutes := s.getTaskTimeoutMinutes(project)
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	s.RegisterCancel(task.ID, cancel)
	defer func() {
		cancel()
		s.cancels.Delete(task.ID)
	}()

	startedAt := time.Now()
	task.Status = model.TaskStatusRunning
	task.StartedAt = &startedAt
	if err := s.tasks.Update(task); err != nil {
		return err
	}
	s.appendLog(task.ID, model.TaskLogLevelInfo, "任务开始执行")

	var cred *model.RepoCredential
	if project.RepoCredentialID != nil {
		cred, _ = s.credentials.GetByID(*project.RepoCredentialID)
	}

	repoDir, err := s.repoManager.EnsureRepo(runCtx, project.ID, project.RepoURL, project.Branch, cred)
	if err != nil {
		s.appendLog(task.ID, model.TaskLogLevelError, "代码仓库同步失败: "+err.Error())
		return s.failTask(task, err)
	}
	s.appendLog(task.ID, model.TaskLogLevelInfo, "代码仓库同步完成")

	// checkout 到目标 commit，使工作目录包含该 commit 的完整代码
	if err := s.repoManager.Checkout(runCtx, repoDir, task.ToCommit); err != nil {
		s.appendLog(task.ID, model.TaskLogLevelError, "代码迁出失败: "+err.Error())
		return s.failTask(task, err)
	}
	s.appendLog(task.ID, model.TaskLogLevelInfo, "已迁出到 commit "+task.ToCommit)

	prompt := modelConfig.Prompt
	if project.CustomPrompt != "" {
		prompt = prompt + "\n\n## 项目补充说明\n\n" + project.CustomPrompt
	}

	if modelConfig.Type == model.ModelTypeClaudeCLI {
		prompt = review.BuildCLIPrompt(prompt, task.FromCommit, task.ToCommit)
	}
	// Agent 路径：变更信息已在 Task 创建时填充，直接使用
	s.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("变更文件: %d 字符, commit 记录: %d 字符", len(task.DiffContent), len(task.CommitMessages)))

	s.appendLog(task.ID, model.TaskLogLevelInfo, "开始调用 "+string(modelConfig.Type))

	var outputChars int64
	onChunk := func(text string) {
		s.cache.AppendResultChunk(task.ID, text)
		n := atomic.AddInt64(&outputChars, int64(len([]rune(text))))
		// 流式过程中按字符数估算 output token（中英混合约 3 字符/token）
		s.cache.UpdateTokens(task.ID, 0, n/3)
	}
	onLog := func(level, msg string) {
		s.appendLog(task.ID, model.TaskLogLevelInfo, msg)
	}

	result, err := s.reviewerFactory(modelConfig).Review(runCtx, review.ReviewParams{
		Prompt:         prompt,
		WorkDir:        repoDir,
		FromCommit:     task.FromCommit,
		ToCommit:       task.ToCommit,
		DiffContent:    task.DiffContent,
		CommitMessages: task.CommitMessages,
		ModelConfig:    modelConfig,
		OnChunk:        onChunk,
		OnLog:          onLog,
		Replace:        s.sensitiveWordReplacer(),
		Restore:        s.sensitiveWordRestorer(),
	})
	if err != nil {
		switch runCtx.Err() {
		case context.DeadlineExceeded:
			s.appendLog(task.ID, model.TaskLogLevelError, fmt.Sprintf("任务超时 (%d 分钟)", timeoutMinutes))
			return s.failTask(task, fmt.Errorf("任务超时"))
		case context.Canceled:
			s.appendLog(task.ID, model.TaskLogLevelInfo, "任务被取消")
			return s.cancelTaskRecord(task)
		default:
			s.appendLog(task.ID, model.TaskLogLevelError, "Review 调用失败: "+err.Error())
			return s.failTask(task, err)
		}
	}

	s.appendLog(task.ID, model.TaskLogLevelInfo, fmt.Sprintf("Review 完成，耗时 %dms", result.DurationMs))

	finishedAt := time.Now()
	task.Status = model.TaskStatusCompleted
	task.Result = result.Content
	task.InputTokens = result.InputTokens
	task.OutputTokens = result.OutputTokens
	task.CacheCreationTokens = result.CacheCreationTokens
	task.CacheReadTokens = result.CacheReadTokens
	task.FinishedAt = &finishedAt
	s.writeReviewFile(task.ProjectID, task)
	if err := s.tasks.Update(task); err != nil {
		return err
	}
	// 推送最终精确 token 数，显式触发 SSE handler 的 done 检测
	s.cache.UpdateTokens(task.ID, task.InputTokens, task.OutputTokens)
	s.cache.SendNotify(task.ID)
	s.cache.RemoveResult(task.ID)
	s.flushLogs(task.ID)

	s.dispatchNotify(task, project)

	// 只有增量 review（from 等于 LastReviewedCommit）才更新 LastReviewedCommit
	if task.FromCommit == project.LastReviewedCommit {
		project.LastReviewedCommit = task.ToCommit
		return s.projects.Update(project)
	}
	return nil
}

func (s *Scheduler) RegisterCancel(taskID int64, cancel func()) {
	s.cancels.Store(taskID, cancel)
}

func (s *Scheduler) CancelTask(taskID int64) error {
	value, ok := s.cancels.Load(taskID)
	if !ok {
		return fmt.Errorf("task %d is not running", taskID)
	}

	value.(func())()
	return nil
}

func (s *Scheduler) getTaskTimeoutMinutes(project *model.Project) int {
	if project.TaskTimeout != nil {
		return *project.TaskTimeout
	}

	value, err := s.globalConfigs.Get(model.GlobalConfigKeyTaskTimeout)
	if err != nil {
		return 30
	}

	timeoutMinutes, err := strconv.Atoi(value)
	if err != nil || timeoutMinutes < 1 {
		return 30
	}
	return timeoutMinutes
}

func (s *Scheduler) failTask(task *model.Task, err error) error {
	finishedAt := time.Now()
	task.Status = model.TaskStatusFailed
	task.ErrorMessage = err.Error()
	task.FinishedAt = &finishedAt
	updateErr := s.tasks.Update(task)
	s.flushLogs(task.ID)

	if project, pErr := s.projects.GetByID(task.ProjectID); pErr == nil {
		s.dispatchNotify(task, project)
	}

	return updateErr
}

func (s *Scheduler) dispatchNotify(task *model.Task, project *model.Project) {
	if s.notifier == nil || s.users == nil || project.CreatedBy == 0 {
		return
	}
	user, err := s.users.GetByID(project.CreatedBy)
	if err != nil || !user.NotifyEnabled {
		return
	}
	go func() {
		if err := s.notifier.Send(task, project, user); err != nil {
			log.Printf("notify task %d: %v", task.ID, err)
		}
	}()
}

func (s *Scheduler) cancelTaskRecord(task *model.Task) error {
	finishedAt := time.Now()
	task.Status = model.TaskStatusCancelled
	task.FinishedAt = &finishedAt
	updateErr := s.tasks.Update(task)
	s.flushLogs(task.ID)
	return updateErr
}

func (s *Scheduler) sensitiveWordReplacer() func(string) string {
	if s.sensitiveWords == nil {
		return nil
	}
	return s.sensitiveWords.Replace
}

func (s *Scheduler) sensitiveWordRestorer() func(string) string {
	if s.sensitiveWords == nil {
		return nil
	}
	return s.sensitiveWords.Restore
}

// writeReviewFile 将 review 结果写入仓库的 .review/ 目录
func (s *Scheduler) writeReviewFile(projectID int64, task *model.Task) {
	repoDir := filepath.Join(s.repoManager.BaseDir(), fmt.Sprintf("%d", projectID))
	reviewDir := filepath.Join(repoDir, ".review")
	if err := os.MkdirAll(reviewDir, 0755); err != nil {
		log.Printf("create .review dir for project %d: %v", projectID, err)
		return
	}

	shortFrom := ""
	if task.FromCommit != "" && len(task.FromCommit) >= 7 {
		shortFrom = task.FromCommit[:7]
	}
	shortTo := task.ToCommit
	if len(shortTo) >= 7 {
		shortTo = shortTo[:7]
	}

	var filename string
	if shortFrom != "" {
		filename = fmt.Sprintf("%s..%s.md", shortFrom, shortTo)
	} else {
		filename = fmt.Sprintf("%s.md", shortTo)
	}

	var b strings.Builder
	b.WriteString("# Code Review: ")
	if shortFrom != "" {
		b.WriteString(shortFrom + "..")
	}
	b.WriteString(shortTo + "\n\n")
	b.WriteString("## Commit Range\n")
	if task.FromCommit != "" {
		b.WriteString(fmt.Sprintf("- From: %s\n", task.FromCommit))
	}
	b.WriteString(fmt.Sprintf("- To: %s\n", task.ToCommit))
	b.WriteString(fmt.Sprintf("- Date: %s\n", task.FinishedAt.Format("2006-01-02")))
	b.WriteString("\n## Review Result\n")
	b.WriteString(task.Result)

	if err := os.WriteFile(filepath.Join(reviewDir, filename), []byte(b.String()), 0644); err != nil {
		log.Printf("write review file for project %d: %v", projectID, err)
	}
}

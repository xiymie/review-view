package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"review-view/internal/model"
	"review-view/internal/review"
	gormstore "review-view/internal/store/gorm"
)

func TestSchedulerReviewWorkflowModeSwitchesToEino(t *testing.T) {
	deps := newSchedulerDeps(t)
	cache := NewTaskCache(deps.stores.Tasks)
	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		cache,
		1,
		time.Second,
	)
	if err := scheduler.SetReviewWorkflowMode("eino"); err != nil {
		t.Fatalf("set workflow mode: %v", err)
	}
	if _, ok := scheduler.reviewWorkflow.(*einoReviewWorkflow); !ok {
		t.Fatalf("expected eino workflow, got %T", scheduler.reviewWorkflow)
	}
}

func TestSchedulerReviewWorkflowModeRejectsUnsupportedValue(t *testing.T) {
	deps := newSchedulerDeps(t)
	scheduler := newTestScheduler(deps)
	if err := scheduler.SetReviewWorkflowMode("unknown"); err == nil {
		t.Fatal("expected unsupported workflow mode error")
	}
}

func TestEinoReviewWorkflowDelegatesToFallback(t *testing.T) {
	deps := newSchedulerDeps(t)
	task := createPendingSchedulerTask(t, deps)
	input := ReviewInput{TaskID: task.ID}
	called := false
	cache := NewTaskCache(deps.stores.Tasks)
	workflow := newEinoReviewWorkflow(ReviewWorkflowFunc(func(_ context.Context, got ReviewInput) error {
		called = true
		if got != input {
			t.Fatalf("unexpected input: %+v", got)
		}
		return nil
	}), cache)

	if err := workflow.Run(context.Background(), input); err != nil {
		t.Fatalf("run eino workflow placeholder: %v", err)
	}
	if !called {
		t.Fatal("expected fallback workflow to be called")
	}

	persistedLogs, err := deps.stores.Tasks.ListLogs(task.ID)
	if err != nil {
		t.Fatalf("list persisted logs: %v", err)
	}
	assertEinoNodeLogs(t, persistedLogs, "prepare", "sync_repo", "checkout", "load_skills", "build_prompt", "run", "sensitive_scan", "persist_result", "finish")
}

func TestEinoReviewWorkflowBuildsPromptNode(t *testing.T) {
	deps := newSchedulerDeps(t)
	deps.project.CustomPrompt = "project-specific rule"
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project prompt: %v", err)
	}

	task := createPendingSchedulerTask(t, deps)
	cache := NewTaskCache(deps.stores.Tasks)
	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		cache,
		1,
		time.Second,
	)
	legacy := newLegacyReviewWorkflow(scheduler)
	workflow := newEinoReviewWorkflow(legacy, cache)

	if err := workflow.Run(context.Background(), ReviewInput{TaskID: task.ID}); err != nil {
		t.Fatalf("run eino workflow: %v", err)
	}

	persistedLogs, err := deps.stores.Tasks.ListLogs(task.ID)
	if err != nil {
		t.Fatalf("list persisted logs: %v", err)
	}
	assertEinoNodeLogs(t, persistedLogs, "build_prompt")
}

func TestEinoReviewWorkflowLoadsStaticSkills(t *testing.T) {
	deps := newSchedulerDeps(t)
	var capturedPrompt string
	deps.reviewerFactory = func(*model.ModelConfig) review.Reviewer {
		return schedulerReviewerFunc(func(_ context.Context, params review.ReviewParams) (*review.ReviewResult, error) {
			capturedPrompt = params.Prompt
			return &review.ReviewResult{Content: "ok", InputTokens: 1, OutputTokens: 2, DurationMs: 1}, nil
		})
	}

	task := createPendingSchedulerTask(t, deps)
	cache := NewTaskCache(deps.stores.Tasks)
	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		cache,
		1,
		time.Second,
	)
	workflow := newEinoReviewWorkflow(newLegacyReviewWorkflow(scheduler), cache)

	if err := workflow.Run(context.Background(), ReviewInput{TaskID: task.ID}); err != nil {
		t.Fatalf("run eino workflow: %v", err)
	}
	assertTaskLogsContain(t, mustListTaskLogs(t, deps, task.ID), "Eino 节点 load_skills 开始", "Eino 节点 load_skills 完成")
	if strings.Contains(capturedPrompt, "## 静态 Review Skills") || strings.Contains(capturedPrompt, "performance-optimizer") || strings.Contains(capturedPrompt, "性能优化专家") {
		t.Fatalf("expected static skills to be disabled by default without ReviewSkillService, got:\n%s", capturedPrompt)
	}
}

func TestEinoReviewWorkflowStreamsChunksToTaskCache(t *testing.T) {
	deps := newSchedulerDeps(t)
	chunked := make(chan struct{})
	release := make(chan struct{})
	deps.reviewerFactory = func(*model.ModelConfig) review.Reviewer {
		return schedulerReviewerFunc(func(_ context.Context, params review.ReviewParams) (*review.ReviewResult, error) {
			params.OnChunk("hello ")
			params.OnChunk("world")
			close(chunked)
			<-release
			return &review.ReviewResult{Content: "final result", InputTokens: 5, OutputTokens: 7, DurationMs: 1}, nil
		})
	}

	task := createPendingSchedulerTask(t, deps)
	cache := NewTaskCache(deps.stores.Tasks)
	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		cache,
		1,
		time.Second,
	)
	workflow := newEinoReviewWorkflow(newLegacyReviewWorkflow(scheduler), cache)
	done := make(chan error, 1)
	go func() {
		done <- workflow.Run(context.Background(), ReviewInput{TaskID: task.ID})
	}()

	select {
	case <-chunked:
	case <-time.After(time.Second):
		t.Fatal("reviewer did not stream chunks")
	}

	if got := cache.GetResult(task.ID); got != "hello world" {
		t.Fatalf("expected streamed result in cache, got %q", got)
	}
	_, outTokens := cache.GetTokens(task.ID)
	if outTokens == 0 {
		t.Fatal("expected streaming token estimate to be updated")
	}

	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run eino workflow with streaming fallback: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("workflow did not finish")
	}
}

func TestEinoReviewWorkflowRunsGitOperationNodes(t *testing.T) {
	deps := newSchedulerDeps(t)
	task := createPendingSchedulerTask(t, deps)
	cache := NewTaskCache(deps.stores.Tasks)
	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		cache,
		1,
		time.Second,
	)
	workflow := newEinoReviewWorkflow(newLegacyReviewWorkflow(scheduler), cache)

	if err := workflow.Run(context.Background(), ReviewInput{TaskID: task.ID}); err != nil {
		t.Fatalf("run eino workflow: %v", err)
	}

	persistedLogs, err := deps.stores.Tasks.ListLogs(task.ID)
	if err != nil {
		t.Fatalf("list persisted logs: %v", err)
	}
	assertEinoNodeLogs(t, persistedLogs, "sync_repo", "checkout")
	assertTaskLogsContain(t, persistedLogs, "代码仓库同步完成", "已迁出到 commit "+task.ToCommit)
}

func TestEinoReviewWorkflowRunsSensitiveScanNode(t *testing.T) {
	deps := newSchedulerDeps(t)
	task := createPendingSchedulerTask(t, deps)
	cache := NewTaskCache(deps.stores.Tasks)
	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		cache,
		1,
		time.Second,
	)
	workflow := newEinoReviewWorkflow(newLegacyReviewWorkflow(scheduler), cache)

	if err := workflow.Run(context.Background(), ReviewInput{TaskID: task.ID}); err != nil {
		t.Fatalf("run eino workflow: %v", err)
	}

	persistedLogs, err := deps.stores.Tasks.ListLogs(task.ID)
	if err != nil {
		t.Fatalf("list persisted logs: %v", err)
	}
	assertEinoNodeLogs(t, persistedLogs, "sensitive_scan")
}

func TestEinoReviewWorkflowRunsPersistResultNode(t *testing.T) {
	deps := newSchedulerDeps(t)
	task := createPendingSchedulerTask(t, deps)
	cache := NewTaskCache(deps.stores.Tasks)
	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		cache,
		1,
		time.Second,
	)
	workflow := newEinoReviewWorkflow(newLegacyReviewWorkflow(scheduler), cache)

	if err := workflow.Run(context.Background(), ReviewInput{TaskID: task.ID}); err != nil {
		t.Fatalf("run eino workflow: %v", err)
	}

	persistedLogs, err := deps.stores.Tasks.ListLogs(task.ID)
	if err != nil {
		t.Fatalf("list persisted logs: %v", err)
	}
	assertEinoNodeLogs(t, persistedLogs, "persist_result")

	storedTask, err := deps.stores.Tasks.GetByID(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != model.TaskStatusCompleted {
		t.Fatalf("expected completed task, got %s", storedTask.Status)
	}
	if storedTask.InputTokens != 11 || storedTask.OutputTokens != 7 {
		t.Fatalf("expected persisted tokens 11/7, got %d/%d", storedTask.InputTokens, storedTask.OutputTokens)
	}
}

func TestRunOnceStartsPendingTasksFIFO(t *testing.T) {
	deps := newSchedulerDeps(t)
	base := time.Now()

	first := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusPending,
		ToCommit:    "b1",
		TriggeredBy: model.TaskTriggeredByManual,
		CreatedAt:   base,
	}
	second := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusPending,
		ToCommit:    "b2",
		TriggeredBy: model.TaskTriggeredByManual,
		CreatedAt:   base.Add(time.Second),
	}
	if err := deps.stores.Tasks.Create(first); err != nil {
		t.Fatalf("create first task: %v", err)
	}
	if err := deps.stores.Tasks.Create(second); err != nil {
		t.Fatalf("create second task: %v", err)
	}

	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		NewTaskCache(deps.stores.Tasks),
		2,
		time.Second,
	)

	var mu sync.Mutex
	started := []int64{}
	done := make(chan struct{}, 2)
	scheduler.onTaskLaunched = func(taskID int64) {
		mu.Lock()
		started = append(started, taskID)
		mu.Unlock()
	}
	scheduler.reviewWorkflow = ReviewWorkflowFunc(func(_ context.Context, _ ReviewInput) error {
		done <- struct{}{}
		return nil
	})

	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}

	<-done
	<-done

	mu.Lock()
	defer mu.Unlock()
	if len(started) != 2 || started[0] != first.ID || started[1] != second.ID {
		t.Fatalf("unexpected execution order: %+v", started)
	}
}

func TestExecutorCompletesTaskAndUpdatesProjectCommit(t *testing.T) {
	deps := newSchedulerDeps(t)

	task := &model.Task{
		ProjectID:      deps.project.ID,
		Status:         model.TaskStatusPending,
		ToCommit:       "b2",
		TriggeredBy:    model.TaskTriggeredByManual,
		DiffContent:    "M\tmain.go\n",
		CommitMessages: "abc1234 init\n",
	}
	if err := deps.stores.Tasks.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		NewTaskCache(deps.stores.Tasks),
		1,
		time.Second,
	)

	if err := scheduler.ExecuteTask(context.Background(), task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	storedTask, err := deps.stores.Tasks.GetByID(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != model.TaskStatusCompleted || storedTask.Result == "" {
		t.Fatalf("unexpected task result %+v", storedTask)
	}

	project, err := deps.stores.Projects.GetByID(deps.project.ID)
	if err != nil {
		t.Fatalf("load project: %v", err)
	}
	if project.LastReviewedCommit != task.ToCommit {
		t.Fatalf("expected project commit updated, got %q", project.LastReviewedCommit)
	}
}

func TestCancelTaskCallsRegisteredCancel(t *testing.T) {
	deps := newSchedulerDeps(t)

	scheduler := NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		NewTaskCache(deps.stores.Tasks),
		1,
		time.Second,
	)

	cancelled := false
	scheduler.RegisterCancel(1, func() { cancelled = true })
	if err := scheduler.CancelTask(1); err != nil {
		t.Fatalf("cancel task: %v", err)
	}
	if !cancelled {
		t.Fatal("expected cancel func to be called")
	}
}

func TestExecutorFailsTaskWhenReviewerReturnsError(t *testing.T) {
	deps := newSchedulerDeps(t)
	reviewErr := errors.New("model unavailable")
	deps.reviewerFactory = func(*model.ModelConfig) review.Reviewer {
		return schedulerReviewerFunc(func(context.Context, review.ReviewParams) (*review.ReviewResult, error) {
			return nil, reviewErr
		})
	}

	task := createPendingSchedulerTask(t, deps)
	scheduler := newTestScheduler(deps)

	err := scheduler.ExecuteTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("execute task should persist failure without returning reviewer error: %v", err)
	}

	storedTask, err := deps.stores.Tasks.GetByID(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != model.TaskStatusFailed {
		t.Fatalf("expected failed task, got %s", storedTask.Status)
	}
	if !strings.Contains(storedTask.ErrorMessage, reviewErr.Error()) {
		t.Fatalf("expected error message to contain %q, got %q", reviewErr.Error(), storedTask.ErrorMessage)
	}
}

func TestExecutorMarksTaskFailedOnTimeout(t *testing.T) {
	deps := newSchedulerDeps(t)
	zeroTimeout := 0
	deps.project.TaskTimeout = &zeroTimeout
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project timeout: %v", err)
	}
	deps.reviewerFactory = func(*model.ModelConfig) review.Reviewer {
		return schedulerReviewerFunc(func(ctx context.Context, _ review.ReviewParams) (*review.ReviewResult, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		})
	}

	task := createPendingSchedulerTask(t, deps)
	scheduler := newTestScheduler(deps)

	if err := scheduler.ExecuteTask(context.Background(), task.ID); err != nil {
		t.Fatalf("execute timeout task: %v", err)
	}

	storedTask, err := deps.stores.Tasks.GetByID(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != model.TaskStatusFailed {
		t.Fatalf("expected failed task, got %s", storedTask.Status)
	}
	if storedTask.ErrorMessage != "任务超时" {
		t.Fatalf("expected timeout error message, got %q", storedTask.ErrorMessage)
	}
}

func TestExecutorMarksTaskCancelledWhenCancelTaskCalled(t *testing.T) {
	deps := newSchedulerDeps(t)
	started := make(chan struct{})
	deps.reviewerFactory = func(*model.ModelConfig) review.Reviewer {
		return schedulerReviewerFunc(func(ctx context.Context, _ review.ReviewParams) (*review.ReviewResult, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		})
	}

	task := createPendingSchedulerTask(t, deps)
	scheduler := newTestScheduler(deps)
	done := make(chan error, 1)
	go func() {
		done <- scheduler.ExecuteTask(context.Background(), task.ID)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("reviewer did not start")
	}
	if err := scheduler.CancelTask(task.ID); err != nil {
		t.Fatalf("cancel task: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("execute cancelled task: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("execute task did not finish after cancellation")
	}

	storedTask, err := deps.stores.Tasks.GetByID(task.ID)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if storedTask.Status != model.TaskStatusCancelled {
		t.Fatalf("expected cancelled task, got %s", storedTask.Status)
	}
}

type schedulerDeps struct {
	stores          gormstore.Stores
	project         *model.Project
	repoManager     *review.RepositoryManager
	reviewerFactory func(*model.ModelConfig) review.Reviewer
}

func assertEinoNodeLogs(t *testing.T, logs []model.TaskLog, nodes ...string) {
	t.Helper()
	messages := make([]string, 0, len(logs))
	for _, log := range logs {
		messages = append(messages, log.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, node := range nodes {
		start := "Eino 节点 " + node + " 开始"
		end := "Eino 节点 " + node + " 完成"
		if !strings.Contains(joined, start) {
			t.Fatalf("expected log %q in logs:\n%s", start, joined)
		}
		if !strings.Contains(joined, end) {
			t.Fatalf("expected log %q in logs:\n%s", end, joined)
		}
	}
}

func assertTaskLogsContain(t *testing.T, logs []model.TaskLog, wants ...string) {
	t.Helper()
	messages := make([]string, 0, len(logs))
	for _, log := range logs {
		messages = append(messages, log.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, want := range wants {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected log %q in logs:\n%s", want, joined)
		}
	}
}

func mustListTaskLogs(t *testing.T, deps schedulerDeps, taskID int64) []model.TaskLog {
	t.Helper()
	logs, err := deps.stores.Tasks.ListLogs(taskID)
	if err != nil {
		t.Fatalf("list task logs: %v", err)
	}
	return logs
}

func newTestScheduler(deps schedulerDeps) *Scheduler {
	return NewScheduler(
		deps.stores.Projects,
		deps.stores.ModelConfigs,
		deps.stores.Tasks,
		deps.stores.GlobalConfigs,
		deps.repoManager,
		deps.stores.Credentials,
		deps.reviewerFactory,
		NewTaskCache(deps.stores.Tasks),
		1,
		time.Second,
	)
}

func createPendingSchedulerTask(t *testing.T, deps schedulerDeps) *model.Task {
	t.Helper()
	task := &model.Task{
		ProjectID:      deps.project.ID,
		Status:         model.TaskStatusPending,
		ToCommit:       "b2",
		TriggeredBy:    model.TaskTriggeredByManual,
		DiffContent:    "M	main.go\n",
		CommitMessages: "abc1234 test\n",
	}
	if err := deps.stores.Tasks.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	return task
}

func newSchedulerDeps(t *testing.T) schedulerDeps {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "review-view-test.db")
	db, err := gormstore.Open("file:" + dbPath + "?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	stores := gormstore.New(db)
	if err := stores.GlobalConfigs.EnsureDefaults(); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	modelConfig := &model.ModelConfig{
		Name:   "openai",
		Type:   model.ModelTypeOpenAI,
		Model:  "gpt-4o-mini",
		Prompt: "review prompt",
	}
	if err := stores.ModelConfigs.Create(modelConfig); err != nil {
		t.Fatalf("create model config: %v", err)
	}

	project := &model.Project{
		Name:             "review-view",
		RepoURL:          "https://example.com/review-view.git",
		Branch:           "main",
		ModelConfigID:    modelConfig.ID,
		OverflowStrategy: model.OverflowStrategyQueue,
	}
	if err := stores.Projects.Create(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	repoManager := review.NewRepositoryManager(t.TempDir(), &schedulerGitRunner{})
	reviewerFactory := func(*model.ModelConfig) review.Reviewer {
		return schedulerReviewer{}
	}

	return schedulerDeps{
		stores:          stores,
		project:         project,
		repoManager:     repoManager,
		reviewerFactory: reviewerFactory,
	}
}

type schedulerGitRunner struct{}

func (schedulerGitRunner) Run(_ context.Context, _ string, _ string, args ...string) (string, error) {
	if len(args) > 0 {
		switch args[0] {
		case "show", "diff":
			return "diff body", nil
		}
	}
	return "", nil
}

type schedulerReviewer struct{}

func (schedulerReviewer) Review(_ context.Context, _ review.ReviewParams) (*review.ReviewResult, error) {
	return &review.ReviewResult{
		Content:      "发现空指针风险",
		InputTokens:  11,
		OutputTokens: 7,
		DurationMs:   3,
	}, nil
}

type schedulerReviewerFunc func(context.Context, review.ReviewParams) (*review.ReviewResult, error)

func (f schedulerReviewerFunc) Review(ctx context.Context, params review.ReviewParams) (*review.ReviewResult, error) {
	return f(ctx, params)
}

func TestExecutorDoesNotUpdateCommitForNonIncrementalRange(t *testing.T) {
	deps := newSchedulerDeps(t)
	deps.project.LastReviewedCommit = "a1"
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project: %v", err)
	}

	task := &model.Task{
		ProjectID:      deps.project.ID,
		Status:         model.TaskStatusPending,
		FromCommit:     "x9",
		ToCommit:       "b2",
		TriggeredBy:    model.TaskTriggeredByManual,
		DiffContent:    "M\tmain.go\n",
		CommitMessages: "abc1234 fix\n",
	}
	if err := deps.stores.Tasks.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	scheduler := NewScheduler(
		deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks,
		deps.stores.GlobalConfigs, deps.repoManager, deps.stores.Credentials, deps.reviewerFactory,
		NewTaskCache(deps.stores.Tasks), 1, time.Second,
	)

	if err := scheduler.ExecuteTask(context.Background(), task.ID); err != nil {
		t.Fatalf("execute task: %v", err)
	}

	project, _ := deps.stores.Projects.GetByID(deps.project.ID)
	if project.LastReviewedCommit != "a1" {
		t.Fatalf("expected LastReviewedCommit unchanged, got %q", project.LastReviewedCommit)
	}
}

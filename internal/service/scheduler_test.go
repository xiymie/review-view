package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"review-view/internal/model"
	"review-view/internal/review"
	gormstore "review-view/internal/store/gorm"
)

func TestRunOnceStartsPendingTasksFIFO(t *testing.T) {
	deps := newSchedulerDeps(t)

	first := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusPending,
		ToCommit:    "b1",
		TriggeredBy: model.TaskTriggeredByManual,
		CreatedAt:   time.Unix(10, 0),
	}
	second := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusPending,
		ToCommit:    "b2",
		TriggeredBy: model.TaskTriggeredByManual,
		CreatedAt:   time.Unix(20, 0),
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
	scheduler.execute = func(_ context.Context, _ int64) error {
		done <- struct{}{}
		return nil
	}

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

func TestCancelTaskMarksTaskCancelled(t *testing.T) {
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

type schedulerDeps struct {
	stores          gormstore.Stores
	project         *model.Project
	repoManager     *review.RepositoryManager
	reviewerFactory func(*model.ModelConfig) review.Reviewer
}

func newSchedulerDeps(t *testing.T) schedulerDeps {
	t.Helper()

	db := gormstore.NewTestDB(t)
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
		Content:     "发现空指针风险",
		InputTokens: 11,
		OutputTokens: 7,
		DurationMs:  3,
	}, nil
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

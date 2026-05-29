package service_test

import (
	"context"
	"testing"

	"review-view/internal/model"
	"review-view/internal/review"
	"review-view/internal/service"
	gormstore "review-view/internal/store/gorm"
)

func TestTriggerSkipsExistingCompletedRange(t *testing.T) {
	deps := newTaskServiceDeps(t)
	deps.project.LastReviewedCommit = "a1"
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project: %v", err)
	}

	completed := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusCompleted,
		FromCommit:  "a1",
		ToCommit:    "b2",
		TriggeredBy: model.TaskTriggeredByManual,
	}
	if err := deps.stores.Tasks.Create(completed); err != nil {
		t.Fatalf("create completed task: %v", err)
	}

	svc := service.NewTaskService(deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks, deps.repoManager, deps.stores.Credentials, nil)
	task, skipped, err := svc.Trigger(context.Background(), service.TriggerInput{
		ProjectID:    deps.project.ID,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: "b2",
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if !skipped || task != nil {
		t.Fatalf("expected skip, got task=%+v skipped=%v", task, skipped)
	}
}

func TestTriggerSkipsExistingActiveRange(t *testing.T) {
	deps := newTaskServiceDeps(t)
	deps.project.LastReviewedCommit = "a1"
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project: %v", err)
	}

	// 创建一个 pending 状态的同范围任务
	pending := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusPending,
		FromCommit:  "a1",
		ToCommit:    "b2",
		TriggeredBy: model.TaskTriggeredByManual,
	}
	if err := deps.stores.Tasks.Create(pending); err != nil {
		t.Fatalf("create pending task: %v", err)
	}

	svc := service.NewTaskService(deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks, deps.repoManager, deps.stores.Credentials, nil)
	task, skipped, err := svc.Trigger(context.Background(), service.TriggerInput{
		ProjectID:    deps.project.ID,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: "b2",
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if !skipped || task != nil {
		t.Fatalf("expected skip due to active task, got task=%+v skipped=%v", task, skipped)
	}
}

func TestTriggerRejectsWhenProjectIsRunning(t *testing.T) {
	deps := newTaskServiceDeps(t)
	deps.project.OverflowStrategy = model.OverflowStrategyReject
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project: %v", err)
	}

	running := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusRunning,
		ToCommit:    "running-commit",
		TriggeredBy: model.TaskTriggeredByManual,
	}
	if err := deps.stores.Tasks.Create(running); err != nil {
		t.Fatalf("create running task: %v", err)
	}

	svc := service.NewTaskService(deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks, deps.repoManager, deps.stores.Credentials, nil)
	task, skipped, err := svc.Trigger(context.Background(), service.TriggerInput{
		ProjectID:    deps.project.ID,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: "b2",
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if skipped || task.Status != model.TaskStatusRejected {
		t.Fatalf("expected rejected task, got %+v", task)
	}
}

func TestRetryCopiesCommitRangeFromFailedTask(t *testing.T) {
	deps := newTaskServiceDeps(t)

	failed := &model.Task{
		ProjectID:   deps.project.ID,
		Status:      model.TaskStatusFailed,
		FromCommit:  "a1",
		ToCommit:    "b2",
		TriggeredBy: model.TaskTriggeredByManual,
	}
	if err := deps.stores.Tasks.Create(failed); err != nil {
		t.Fatalf("create failed task: %v", err)
	}

	svc := service.NewTaskService(deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks, deps.repoManager, deps.stores.Credentials, nil)
	task, err := svc.Retry(context.Background(), failed.ID)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if task.Status != model.TaskStatusPending || task.ToCommit != "b2" {
		t.Fatalf("unexpected retry task %+v", task)
	}
}

type taskServiceDeps struct {
	stores      gormstore.Stores
	project     *model.Project
	repoManager *review.RepositoryManager
}

func newTaskServiceDeps(t *testing.T) taskServiceDeps {
	t.Helper()

	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	project := &model.Project{
		Name:             "review-view",
		RepoURL:          "https://example.com/review-view.git",
		Branch:           "main",
		ModelConfigID:    1,
		OverflowStrategy: model.OverflowStrategyQueue,
		Status:           model.ProjectStatusReady,
	}
	if err := stores.Projects.Create(project); err != nil {
		t.Fatalf("create project: %v", err)
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

	repoManager := review.NewRepositoryManager(t.TempDir(), &fakeGitRunner{output: "head123\n"})
	return taskServiceDeps{
		stores:      stores,
		project:     project,
		repoManager: repoManager,
	}
}

type fakeGitRunner struct {
	output string
	err    error
}

func (f *fakeGitRunner) Run(_ context.Context, _ string, _ string, _ ...string) (string, error) {
	return f.output, f.err
}

func TestTriggerPopulatesTaskChanges(t *testing.T) {
	deps := newTaskServiceDeps(t)

	runner := &multiResponseGitRunner{
		responses: map[string]string{
			"show":               "M\tmain.go\nA\thelper.go\n",
			"log --oneline":      "abc1234 fix login\n",
		},
	}
	repoManager := review.NewRepositoryManager(t.TempDir(), runner)

	svc := service.NewTaskService(deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks, repoManager, deps.stores.Credentials, nil)
	task, skipped, err := svc.Trigger(context.Background(), service.TriggerInput{
		ProjectID:    deps.project.ID,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: "b2",
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if skipped || task == nil {
		t.Fatalf("expected task created, got skipped=%v", skipped)
	}
	if task.DiffContent != "M\tmain.go\nA\thelper.go\n" {
		t.Fatalf("expected diff name-status in task, got %q", task.DiffContent)
	}
	if task.CommitMessages != "abc1234 fix login\n" {
		t.Fatalf("expected commit messages in task, got %q", task.CommitMessages)
	}
}

// multiResponseGitRunner 根据命令参数返回不同结果
type multiResponseGitRunner struct {
	responses map[string]string
	err       error
}

func (m *multiResponseGitRunner) Run(_ context.Context, _ string, _ string, args ...string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	key := args[0]
	if len(args) > 1 {
		if args[0] == "diff" || args[0] == "log" {
			key = args[0] + " " + args[1]
		}
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return "", nil
}

func TestTriggerRejectsWhenProjectNotReady(t *testing.T) {
	deps := newTaskServiceDeps(t)
	deps.project.Status = model.ProjectStatusInitializing
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project: %v", err)
	}

	svc := service.NewTaskService(deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks, deps.repoManager, deps.stores.Credentials, nil)
	_, _, err := svc.Trigger(context.Background(), service.TriggerInput{
		ProjectID:    deps.project.ID,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: "b2",
	})
	if err == nil {
		t.Fatal("expected error when project is not ready")
	}
}

func TestTriggerUsesExplicitFromCommit(t *testing.T) {
	deps := newTaskServiceDeps(t)
	deps.project.LastReviewedCommit = "a1"
	if err := deps.stores.Projects.Update(deps.project); err != nil {
		t.Fatalf("update project: %v", err)
	}

	runner := &multiResponseGitRunner{
		responses: map[string]string{
			"diff --name-status": "M\tmain.go\n",
			"log --oneline":     "abc1234 fix\n",
		},
	}
	repoManager := review.NewRepositoryManager(t.TempDir(), runner)

	svc := service.NewTaskService(deps.stores.Projects, deps.stores.ModelConfigs, deps.stores.Tasks, repoManager, deps.stores.Credentials, nil)
	task, skipped, err := svc.Trigger(context.Background(), service.TriggerInput{
		ProjectID:    deps.project.ID,
		TriggeredBy:  model.TaskTriggeredByManual,
		TargetCommit: "b2",
		FromCommit:   "x9",
	})
	if err != nil {
		t.Fatalf("trigger: %v", err)
	}
	if skipped || task == nil {
		t.Fatalf("expected task created")
	}
	if task.FromCommit != "x9" {
		t.Fatalf("expected from_commit=x9, got %q", task.FromCommit)
	}
}

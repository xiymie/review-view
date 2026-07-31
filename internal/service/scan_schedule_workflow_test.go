package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"review-view/internal/model"
	"review-view/internal/review"
	gormstore "review-view/internal/store/gorm"
)

func TestScanScheduleWorkflowCompletesChangedBranch(t *testing.T) {
	stores := newScanWorkflowStores(t)
	modelConfig := createScanWorkflowModelConfig(t, stores)
	schedule := &model.ScanSchedule{
		Name:          "daily scan",
		RepoURL:       "https://example.com/repo.git",
		ModelConfigID: modelConfig.ID,
		Enabled:       true,
	}
	if err := stores.ScanSchedules.Create(schedule); err != nil {
		t.Fatalf("create scan schedule: %v", err)
	}

	var preparedRepoDir string
	var uploadedReport string
	cleanupCalled := false
	svc := NewScanService(
		stores.ScanSchedules,
		stores.ScanJobs,
		stores.ModelConfigs,
		stores.Credentials,
		NewSettingsService(stores.GlobalConfigs),
		review.NewRepositoryManager(t.TempDir(), nil),
	)
	svc.repoPreparer = func(_ context.Context, repoDir string, cfg *ScanConfig) error {
		preparedRepoDir = repoDir
		if cfg.RepoURL != schedule.RepoURL {
			t.Fatalf("unexpected repo url %q", cfg.RepoURL)
		}
		return nil
	}
	svc.branchLister = func(_ context.Context, repoDir string) ([]string, error) {
		if repoDir != preparedRepoDir {
			t.Fatalf("branch lister repoDir=%q, want %q", repoDir, preparedRepoDir)
		}
		return []string{"main"}, nil
	}
	svc.branchHeadGetter = func(_ context.Context, _ string, branch string) (string, error) {
		if branch != "main" {
			t.Fatalf("unexpected branch %q", branch)
		}
		return "head-main", nil
	}
	svc.recentCommitsGetter = func(_ context.Context, _ string, branch string, n int) ([]commitEntry, error) {
		if branch != "main" || n != 3 {
			t.Fatalf("unexpected recent commits request branch=%q n=%d", branch, n)
		}
		return []commitEntry{
			{Hash: "head-main", Message: "change config", Author: "Alice", Time: "2026-07-30"},
			{Hash: "base-main", Message: "base", Author: "Bob", Time: "2026-07-29"},
		}, nil
	}
	svc.diffLoader = func(_ context.Context, _ string, branch, fromCommit string) (string, error) {
		if branch != "main" || fromCommit != "base-main" {
			t.Fatalf("unexpected diff request branch=%q from=%q", branch, fromCommit)
		}
		return "diff --git a/app.yaml b/app.yaml", nil
	}
	svc.llmCaller = func(_ context.Context, _ *model.ModelConfig, prompt string) (string, error) {
		for _, want := range []string{"https://example.com/repo.git", "main", "change config", "diff --git"} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("expected prompt to contain %q, got:\n%s", want, prompt)
			}
		}
		return "**风险等级**：低风险\n**命中风险类型**：CFG", nil
	}
	svc.reportUploader = func(_ *ScanConfig, name, content string) (string, error) {
		if name != schedule.Name {
			t.Fatalf("unexpected report name %q", name)
		}
		if !strings.Contains(content, "## 分支: main") || !strings.Contains(content, "低风险") {
			t.Fatalf("unexpected report content:\n%s", content)
		}
		uploadedReport = content
		return "nas://report.md", nil
	}
	svc.oldReportsCleaner = func(_ context.Context, scheduleID int64, _ *ScanConfig) {
		if scheduleID != schedule.ID {
			t.Fatalf("unexpected cleanup schedule id %d", scheduleID)
		}
		cleanupCalled = true
	}

	if err := svc.RunSchedule(context.Background(), schedule.ID); err != nil {
		t.Fatalf("run schedule workflow: %v", err)
	}
	if uploadedReport == "" {
		t.Fatal("expected report to be uploaded")
	}
	if !cleanupCalled {
		t.Fatal("expected old report cleanup to be called")
	}

	jobs, err := stores.ScanJobs.ListBySchedule(schedule.ID, 10)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	job := jobs[0]
	if job.Status != model.ScanJobStatusCompleted {
		t.Fatalf("expected completed job, got %s", job.Status)
	}
	if job.BranchCount != 1 || job.ChangedBranchCount != 1 || job.RiskCount != 1 {
		t.Fatalf("unexpected branch counts: total=%d changed=%d risk=%d", job.BranchCount, job.ChangedBranchCount, job.RiskCount)
	}
	if job.ReportPath != "nas://report.md" {
		t.Fatalf("unexpected report path %q", job.ReportPath)
	}

	results, err := stores.ScanJobs.ListBranchResults(job.ID)
	if err != nil {
		t.Fatalf("list branch results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 branch result, got %d", len(results))
	}
	if results[0].BranchName != "main" || results[0].RiskLevel != "low" || !results[0].HasRisk {
		t.Fatalf("unexpected branch result: %+v", results[0])
	}

	logs, err := stores.ScanJobs.ListJobLogs(job.ID)
	if err != nil {
		t.Fatalf("list job logs: %v", err)
	}
	assertScanNodeLogs(t, logs, "create_job", "build_config", "prepare_repo", "list_branches", "process_branches", "finalize")
	assertScanBranchNodeLogs(t, logs, "main", "prepare_commits", "load_diff", "build_prompt", "call_llm", "build_result")

	storedSchedule, err := stores.ScanSchedules.GetByID(schedule.ID)
	if err != nil {
		t.Fatalf("load schedule: %v", err)
	}
	var checkpoints map[string]string
	if err := json.Unmarshal([]byte(storedSchedule.BranchCheckpoints), &checkpoints); err != nil {
		t.Fatalf("unmarshal checkpoints: %v", err)
	}
	if checkpoints["main"] != "head-main" {
		t.Fatalf("expected main checkpoint head-main, got %q", checkpoints["main"])
	}
}

func TestScanScheduleWorkflowFailsJobWhenBuildConfigFails(t *testing.T) {
	stores := newScanWorkflowStores(t)
	schedule := &model.ScanSchedule{
		Name:          "bad scan",
		RepoURL:       "https://example.com/repo.git",
		ModelConfigID: 999999,
		Enabled:       true,
	}
	if err := stores.ScanSchedules.Create(schedule); err != nil {
		t.Fatalf("create scan schedule: %v", err)
	}
	svc := NewScanService(
		stores.ScanSchedules,
		stores.ScanJobs,
		stores.ModelConfigs,
		stores.Credentials,
		NewSettingsService(stores.GlobalConfigs),
		review.NewRepositoryManager(t.TempDir(), nil),
	)

	err := svc.RunSchedule(context.Background(), schedule.ID)
	if err == nil {
		t.Fatal("expected build config failure")
	}
	if !strings.Contains(err.Error(), "build config") {
		t.Fatalf("expected build config error, got %v", err)
	}

	jobs, err := stores.ScanJobs.ListBySchedule(schedule.ID, 10)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Status != model.ScanJobStatusFailed {
		t.Fatalf("expected failed job, got %s", jobs[0].Status)
	}
	if !strings.Contains(jobs[0].ErrorMessage, "build config") {
		t.Fatalf("expected job error to contain build config, got %q", jobs[0].ErrorMessage)
	}
}

func assertScanNodeLogs(t *testing.T, logs []model.ScanJobLog, nodes ...string) {
	t.Helper()
	messages := make([]string, 0, len(logs))
	for _, log := range logs {
		messages = append(messages, log.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, node := range nodes {
		start := "Scan 节点 " + node + " 开始"
		end := "Scan 节点 " + node + " 完成"
		if !strings.Contains(joined, start) {
			t.Fatalf("expected log %q in logs:\n%s", start, joined)
		}
		if !strings.Contains(joined, end) {
			t.Fatalf("expected log %q in logs:\n%s", end, joined)
		}
	}
}

func assertScanBranchNodeLogs(t *testing.T, logs []model.ScanJobLog, branch string, nodes ...string) {
	t.Helper()
	messages := make([]string, 0, len(logs))
	for _, log := range logs {
		messages = append(messages, log.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, node := range nodes {
		start := "Scan 分支 " + branch + " 节点 " + node + " 开始"
		end := "Scan 分支 " + branch + " 节点 " + node + " 完成"
		if !strings.Contains(joined, start) {
			t.Fatalf("expected log %q in logs:\n%s", start, joined)
		}
		if !strings.Contains(joined, end) {
			t.Fatalf("expected log %q in logs:\n%s", end, joined)
		}
	}
}

func newScanWorkflowStores(t *testing.T) gormstore.Stores {
	t.Helper()
	db, err := gormstore.Open("file:" + t.TempDir() + "/scan-workflow.db?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	stores := gormstore.New(db)
	if err := stores.GlobalConfigs.EnsureDefaults(); err != nil {
		t.Fatalf("ensure global config defaults: %v", err)
	}
	return stores
}

func createScanWorkflowModelConfig(t *testing.T, stores gormstore.Stores) *model.ModelConfig {
	t.Helper()
	modelConfig := &model.ModelConfig{
		Name:   "scan-model",
		Type:   model.ModelTypeOpenAI,
		Model:  "fake-model",
		Prompt: "unused",
	}
	if err := stores.ModelConfigs.Create(modelConfig); err != nil {
		t.Fatalf("create model config: %v", err)
	}
	return modelConfig
}

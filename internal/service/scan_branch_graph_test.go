package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"review-view/internal/model"
)

func TestScanBranchGraphBuildsResultWithDiffPromptAndRisk(t *testing.T) {
	var capturedPrompt string
	svc := &ScanService{
		llmCaller: func(_ context.Context, _ *model.ModelConfig, prompt string) (string, error) {
			capturedPrompt = prompt
			return "**风险等级**：高风险\n**命中风险类型**：DB", nil
		},
		diffLoader: func(_ context.Context, repoDir, branch, fromCommit string) (string, error) {
			if repoDir != "/tmp/repo" {
				t.Fatalf("unexpected repoDir %q", repoDir)
			}
			if branch != "main" {
				t.Fatalf("unexpected branch %q", branch)
			}
			if fromCommit != "aaa111" {
				t.Fatalf("unexpected fromCommit %q", fromCommit)
			}
			return "diff --git a/main.go b/main.go", nil
		},
	}
	commits := []commitEntry{
		{Hash: "bbb222", Message: "add payment check", Author: "Alice", Time: "2026-07-30 10:00:00 +0800"},
		{Hash: "aaa111", Message: "init payment", Author: "Bob", Time: "2026-07-29 10:00:00 +0800"},
	}

	result, err := newScanBranchGraph(svc).Run(context.Background(), scanBranchGraphInput{
		Config: &ScanConfig{
			RepoURL:     "https://example.com/repo.git",
			ModelConfig: &model.ModelConfig{Model: "fake-model"},
			Prompt:      defaultScanPrompt,
		},
		RepoDir: "/tmp/repo",
		Branch:  "main",
		Commits: commits,
		RepoURL: "https://example.com/repo.git",
		JobID:   99,
	})
	if err != nil {
		t.Fatalf("run scan branch graph: %v", err)
	}

	if result.JobID != 99 || result.BranchName != "main" {
		t.Fatalf("unexpected result identity: %+v", result)
	}
	if result.FromCommit != "aaa111" || result.ToCommit != "bbb222" {
		t.Fatalf("unexpected commit range: %s..%s", result.FromCommit, result.ToCommit)
	}
	if result.CommitCount != 2 {
		t.Fatalf("expected 2 commits, got %d", result.CommitCount)
	}
	if result.AnalysisStage != model.ScanAnalysisStageWithDiff {
		t.Fatalf("expected with_diff stage, got %s", result.AnalysisStage)
	}
	if !result.HasRisk || result.RiskLevel != "high" {
		t.Fatalf("expected high risk, got hasRisk=%v level=%s", result.HasRisk, result.RiskLevel)
	}

	var gotCommits []commitEntry
	if err := json.Unmarshal([]byte(result.CommitMessages), &gotCommits); err != nil {
		t.Fatalf("unmarshal commit messages: %v", err)
	}
	if len(gotCommits) != 2 || gotCommits[0].Hash != "bbb222" || gotCommits[1].Hash != "aaa111" {
		t.Fatalf("unexpected commits json: %+v", gotCommits)
	}
	for _, want := range []string{"仓库 https://example.com/repo.git", "分支 main", "add payment check", "diff --git a/main.go b/main.go", "高危验证", "CLAUDE.md"} {
		if !strings.Contains(capturedPrompt, want) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", want, capturedPrompt)
		}
	}
}

func TestScanBranchGraphReadsClaudeMDIntoPrompt(t *testing.T) {
	var capturedPrompt string
	repoDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(repoDir, "CLAUDE.md"), []byte("重点检查：订单状态机必须保持幂等，不要把已保护的内部函数误判为高危。"), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	svc := &ScanService{
		llmCaller: func(_ context.Context, _ *model.ModelConfig, prompt string) (string, error) {
			capturedPrompt = prompt
			return "**风险等级**：无风险", nil
		},
		diffLoader: func(context.Context, string, string, string) (string, error) {
			return "diff --git a/order.go b/order.go", nil
		},
	}
	_, err := newScanBranchGraph(svc).Run(context.Background(), scanBranchGraphInput{
		Config:  &ScanConfig{ModelConfig: &model.ModelConfig{Model: "fake-model"}},
		RepoDir: repoDir,
		Branch:  "main",
		Commits: []commitEntry{{Hash: "abc123", Message: "update order", Author: "Alice", Time: "2026-07-30"}},
		RepoURL: "https://example.com/repo.git",
	})
	if err != nil {
		t.Fatalf("run scan branch graph: %v", err)
	}
	for _, want := range []string{"## 项目 CLAUDE.md 细则", "订单状态机必须保持幂等", "不要把已保护的内部函数误判为高危"} {
		if !strings.Contains(capturedPrompt, want) {
			t.Fatalf("expected prompt to contain %q, got:\n%s", want, capturedPrompt)
		}
	}
}

func TestScanBranchGraphFallsBackToDefaultPrompt(t *testing.T) {
	var capturedPrompt string
	svc := &ScanService{
		llmCaller: func(_ context.Context, _ *model.ModelConfig, prompt string) (string, error) {
			capturedPrompt = prompt
			return "**风险等级**：无风险", nil
		},
		diffLoader: func(context.Context, string, string, string) (string, error) {
			return "（无可用 diff）", nil
		},
	}

	result, err := newScanBranchGraph(svc).Run(context.Background(), scanBranchGraphInput{
		Config: &ScanConfig{ModelConfig: &model.ModelConfig{Model: "fake-model"}},
		Branch: "dev",
		Commits: []commitEntry{
			{Hash: "cccc333", Message: "docs", Author: "Alice", Time: "2026-07-30"},
		},
		RepoURL: "https://example.com/repo.git",
		JobID:   100,
	})
	if err != nil {
		t.Fatalf("run scan branch graph: %v", err)
	}
	if result.HasRisk || result.RiskLevel != "none" {
		t.Fatalf("expected none risk, got hasRisk=%v level=%s", result.HasRisk, result.RiskLevel)
	}
	if !strings.Contains(capturedPrompt, "高风险判定前必须做上下文复核") {
		t.Fatalf("expected updated default scan prompt, got:\n%s", capturedPrompt)
	}
}

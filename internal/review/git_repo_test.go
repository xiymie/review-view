package review_test

import (
	"context"
	"errors"
	"testing"

	"review-view/internal/review"
)

func TestEnsureRepoClonesMissingRepository(t *testing.T) {
	runner := &fakeGitRunner{}
	manager := review.NewRepositoryManager("/tmp/repos", runner)

	_, err := manager.EnsureRepo(context.Background(), 42, "https://example.com/review-view.git", "main", nil)
	if err != nil {
		t.Fatalf("ensure repo: %v", err)
	}

	if !runner.HasCall("git", "clone", "--branch", "main", "https://example.com/review-view.git", "/tmp/repos/42") {
		t.Fatalf("expected clone call, got %+v", runner.calls)
	}
}

func TestBuildDiffUsesShowOnFirstReview(t *testing.T) {
	runner := &fakeGitRunner{output: "diff body"}
	manager := review.NewRepositoryManager("/tmp/repos", runner)

	diff, err := manager.BuildDiff(context.Background(), "/tmp/repos/42", "", "abc123")
	if err != nil {
		t.Fatalf("build diff: %v", err)
	}
	if diff != "diff body" {
		t.Fatalf("unexpected diff %q", diff)
	}
	if !runner.HasCall("git", "show", "abc123") {
		t.Fatalf("expected git show, got %+v", runner.calls)
	}
}

func TestResolveHeadCommitFetchesOriginBranch(t *testing.T) {
	runner := &fakeGitRunner{output: "deadbeef\n"}
	manager := review.NewRepositoryManager("/tmp/repos", runner)

	commit, err := manager.ResolveHeadCommit(context.Background(), "/tmp/repos/42", "main")
	if err != nil {
		t.Fatalf("resolve head: %v", err)
	}
	if commit != "deadbeef" {
		t.Fatalf("unexpected commit %q", commit)
	}
}

type fakeGitRunner struct {
	calls  [][]string
	output string
	err    error
}

func (f *fakeGitRunner) Run(_ context.Context, _ string, name string, args ...string) (string, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if f.err != nil {
		return "", f.err
	}
	return f.output, nil
}

func (f *fakeGitRunner) HasCall(parts ...string) bool {
	for _, call := range f.calls {
		if len(call) != len(parts) {
			continue
		}
		match := true
		for i := range parts {
			if call[i] != parts[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

var _ error = errors.New("unused guard")

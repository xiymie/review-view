package gormstore_test

import (
	"testing"
	"time"

	"review-view/internal/model"
	gormstore "review-view/internal/store/gorm"
)

func TestProjectStoreCreateAndList(t *testing.T) {
	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	project := &model.Project{
		Name:             "review-view",
		RepoURL:          "https://example.com/review-view.git",
		Branch:           "main",
		ModelConfigID:    1,
		OverflowStrategy: model.OverflowStrategyQueue,
	}
	if err := stores.Projects.Create(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	items, err := stores.Projects.List()
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(items) != 1 || items[0].Name != "review-view" {
		t.Fatalf("unexpected projects: %+v", items)
	}
}

func TestTaskStorePendingFIFO(t *testing.T) {
	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	now := time.Now()
	first := &model.Task{ProjectID: 1, Status: model.TaskStatusPending, ToCommit: "b1", CreatedAt: now}
	second := &model.Task{ProjectID: 1, Status: model.TaskStatusPending, ToCommit: "b2", CreatedAt: now.Add(time.Second)}

	if err := stores.Tasks.Create(first); err != nil {
		t.Fatal(err)
	}
	if err := stores.Tasks.Create(second); err != nil {
		t.Fatal(err)
	}

	items, err := stores.Tasks.ListPending(10)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(items) != 2 || items[0].ToCommit != "b1" {
		t.Fatalf("expected FIFO order, got %+v", items)
	}
}

func TestGlobalConfigStoreSeedsDefaults(t *testing.T) {
	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	if err := stores.GlobalConfigs.EnsureDefaults(); err != nil {
		t.Fatalf("ensure defaults: %v", err)
	}
	value, err := stores.GlobalConfigs.Get(model.GlobalConfigKeyTaskTimeout)
	if err != nil {
		t.Fatalf("get task timeout: %v", err)
	}
	if value != "30" {
		t.Fatalf("expected default timeout 30, got %q", value)
	}
}

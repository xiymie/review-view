package gormstore_test

import (
	"fmt"
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

// TestScanPromptMigration covers the three EnsureDefaults upgrade paths for the
// scan prompt key: fresh (no row), stale-empty, and legacy default text.
func TestScanPromptMigration(t *testing.T) {
	cases := []struct {
		name         string
		seed         *string // nil = no row; &"" = empty; &legacy = legacy text
		wantCurrent  bool    // true → expect model.DefaultScanPrompt
		wantPreserve bool    // true → expect the seeded value to be unchanged
	}{
		{
			name:        "fresh install seeds DefaultScanPrompt",
			seed:        nil,
			wantCurrent: true,
		},
		{
			name:        "empty value upgraded to DefaultScanPrompt",
			seed:        strPtr(""),
			wantCurrent: true,
		},
		{
			name:        "legacy default upgraded to DefaultScanPrompt",
			seed:        strPtr(model.LegacyDefaultScanPrompt),
			wantCurrent: true,
		},
		{
			name:         "custom value preserved",
			seed:         strPtr("my custom prompt"),
			wantPreserve: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each sub-test gets its own isolated in-memory database so rows
			// don't bleed between cases.
			db, err := gormstore.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name()))
			if err != nil {
				t.Fatalf("open test db: %v", err)
			}
			stores := gormstore.New(db)

			if tc.seed != nil {
				if err := stores.GlobalConfigs.Set(model.GlobalConfigKeyScanPrompt, *tc.seed); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}

			if err := stores.GlobalConfigs.EnsureDefaults(); err != nil {
				t.Fatalf("EnsureDefaults: %v", err)
			}

			got, err := stores.GlobalConfigs.Get(model.GlobalConfigKeyScanPrompt)
			if err != nil {
				t.Fatalf("get: %v", err)
			}

			switch {
			case tc.wantCurrent:
				if got != model.DefaultScanPrompt {
					t.Errorf("expected DefaultScanPrompt, got %q", got[:min(80, len(got))])
				}
			case tc.wantPreserve:
				if got != *tc.seed {
					t.Errorf("expected seeded value %q preserved, got %q", *tc.seed, got)
				}
			}
		})
	}
}

func TestProjectStoreSetAndListSkills(t *testing.T) {
	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	project := &model.Project{
		Name:             "skill-test",
		RepoURL:          "https://example.com/x.git",
		Branch:           "main",
		ModelConfigID:    1,
		OverflowStrategy: model.OverflowStrategyQueue,
	}
	if err := stores.Projects.Create(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// empty initially
	ids, err := stores.Projects.ListSkillIDs(project.ID)
	if err != nil {
		t.Fatalf("list skill ids: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty, got %v", ids)
	}

	// set two skills
	if err := stores.Projects.SetSkills(project.ID, []int64{10, 20}); err != nil {
		t.Fatalf("set skills: %v", err)
	}
	ids, err = stores.Projects.ListSkillIDs(project.ID)
	if err != nil {
		t.Fatalf("list skill ids after set: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 skill ids, got %v", ids)
	}

	// replace with one skill
	if err := stores.Projects.SetSkills(project.ID, []int64{30}); err != nil {
		t.Fatalf("replace skills: %v", err)
	}
	ids, err = stores.Projects.ListSkillIDs(project.ID)
	if err != nil {
		t.Fatalf("list skill ids after replace: %v", err)
	}
	if len(ids) != 1 || ids[0] != 30 {
		t.Fatalf("expected [30], got %v", ids)
	}

	// clear all
	if err := stores.Projects.SetSkills(project.ID, []int64{}); err != nil {
		t.Fatalf("clear skills: %v", err)
	}
	ids, err = stores.Projects.ListSkillIDs(project.ID)
	if err != nil {
		t.Fatalf("list skill ids after clear: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("expected empty after clear, got %v", ids)
	}
}

func strPtr(s string) *string { return &s }

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestScanJobStoreHasRunningBySchedule(t *testing.T) {
	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	running, err := stores.ScanJobs.HasRunningBySchedule(100)
	if err != nil {
		t.Fatalf("has running initial: %v", err)
	}
	if running {
		t.Fatal("expected no running job initially")
	}

	job := &model.ScanJob{ScheduleID: 100, Status: model.ScanJobStatusRunning, TriggeredAt: time.Now()}
	if err := stores.ScanJobs.Create(job); err != nil {
		t.Fatalf("create running job: %v", err)
	}
	running, err = stores.ScanJobs.HasRunningBySchedule(100)
	if err != nil {
		t.Fatalf("has running after create: %v", err)
	}
	if !running {
		t.Fatal("expected running job")
	}

	job.Status = model.ScanJobStatusCompleted
	if err := stores.ScanJobs.Update(job); err != nil {
		t.Fatalf("update completed job: %v", err)
	}
	running, err = stores.ScanJobs.HasRunningBySchedule(100)
	if err != nil {
		t.Fatalf("has running after complete: %v", err)
	}
	if running {
		t.Fatal("expected no running job after complete")
	}
}

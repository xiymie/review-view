package handler

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"review-view/internal/model"
	"review-view/internal/review"
	"review-view/internal/service"
	gormstore "review-view/internal/store/gorm"
)

func NewTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)
	if err := stores.GlobalConfigs.EnsureDefaults(); err != nil {
		t.Fatalf("seed defaults: %v", err)
	}

	modelConfig := &model.ModelConfig{
		Name:   "OpenAI",
		Type:   model.ModelTypeOpenAI,
		Model:  "gpt-4o-mini",
		Prompt: "review prompt",
	}
	if err := stores.ModelConfigs.Create(modelConfig); err != nil {
		t.Fatalf("seed model config: %v", err)
	}
	project := &model.Project{
		Name:             "review-view",
		RepoURL:          "https://example.com/review-view.git",
		Branch:           "main",
		ModelConfigID:    modelConfig.ID,
		OverflowStrategy: model.OverflowStrategyQueue,
		Status:           model.ProjectStatusReady,
	}
	if err := stores.Projects.Create(project); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	repoManager := review.NewRepositoryManager(t.TempDir(), testGitRunner{})
	projectService := service.NewProjectService(stores.Projects, stores.ModelConfigs, stores.Tasks, repoManager, stores.Credentials)
	modelService := service.NewModelConfigService(stores.ModelConfigs)
	settingsService := service.NewSettingsService(stores.GlobalConfigs)
	dashboardService := service.NewDashboardService(stores.Projects, stores.Tasks)
	credentialService := service.NewRepoCredentialService(stores.Credentials, stores.Projects)
	taskService := service.NewTaskService(stores.Projects, stores.ModelConfigs, stores.Tasks, repoManager, stores.Credentials, nil)
	scheduler := service.NewScheduler(stores.Projects, stores.ModelConfigs, stores.Tasks, stores.GlobalConfigs, repoManager, stores.Credentials, nil, service.NewTaskCache(stores.Tasks), 1, 5*time.Second)
	taskHandler := NewTaskHandler(stores.Tasks, stores.Projects, taskService, scheduler, service.NewTaskCache(stores.Tasks), stores.Users)

	return NewRouter(&Handlers{
		Dashboard:   NewDashboardHandler(dashboardService, stores.Users),
		Projects:    NewProjectHandler(projectService, modelService, taskService, stores.Tasks, credentialService, stores.Users),
		Models:      NewModelHandler(modelService),
		Settings:    NewSettingsHandler(settingsService),
		Tasks:       taskHandler,
		Webhook:     NewWebhookHandler(taskService),
		Credentials: NewCredentialHandler(credentialService, stores.Users),
	})
}

type testGitRunner struct{}

func (testGitRunner) Run(_ context.Context, _ string, _ string, args ...string) (string, error) {
	if len(args) > 0 {
		switch args[0] {
		case "show", "diff", "rev-parse":
			return "abc123\n", nil
		case "log":
			return "abc123def4567890|init commit|2026-01-01 00:00:00 +0800|test\n", nil
		}
	}
	return "", nil
}

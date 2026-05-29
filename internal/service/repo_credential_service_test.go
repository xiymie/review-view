package service_test

import (
	"testing"

	"review-view/internal/model"
	"review-view/internal/service"
	gormstore "review-view/internal/store/gorm"
)

func TestCredentialDeleteRejectedWhenReferenced(t *testing.T) {
	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	cred := &model.RepoCredential{Name: "test", Username: "u", Password: "p"}
	if err := stores.Credentials.Create(cred); err != nil {
		t.Fatalf("create credential: %v", err)
	}

	modelConfig := &model.ModelConfig{Name: "m", Type: model.ModelTypeOpenAI, Model: "gpt-4o", Prompt: "p"}
	if err := stores.ModelConfigs.Create(modelConfig); err != nil {
		t.Fatalf("create model config: %v", err)
	}

	project := &model.Project{
		Name:             "proj",
		RepoURL:          "https://example.com/repo.git",
		Branch:           "main",
		ModelConfigID:    modelConfig.ID,
		OverflowStrategy: model.OverflowStrategyQueue,
		Status:           model.ProjectStatusReady,
		RepoCredentialID: &cred.ID,
	}
	if err := stores.Projects.Create(project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	svc := service.NewRepoCredentialService(stores.Credentials, stores.Projects)
	err := svc.Delete(cred.ID)
	if err == nil {
		t.Fatal("expected error when deleting referenced credential")
	}
}

func TestCredentialDeleteSucceedsWhenNotReferenced(t *testing.T) {
	db := gormstore.NewTestDB(t)
	stores := gormstore.New(db)

	cred := &model.RepoCredential{Name: "test", Username: "u", Password: "p"}
	if err := stores.Credentials.Create(cred); err != nil {
		t.Fatalf("create credential: %v", err)
	}

	svc := service.NewRepoCredentialService(stores.Credentials, stores.Projects)
	if err := svc.Delete(cred.ID); err != nil {
		t.Fatalf("delete should succeed: %v", err)
	}
}

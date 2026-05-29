package service

import (
	"fmt"

	"review-view/internal/model"
	"review-view/internal/store"
)

type RepoCredentialService struct {
	credentials store.RepoCredentialStore
	projects    store.ProjectStore
}

func NewRepoCredentialService(credentials store.RepoCredentialStore, projects store.ProjectStore) *RepoCredentialService {
	return &RepoCredentialService{credentials: credentials, projects: projects}
}

func (s *RepoCredentialService) List() ([]model.RepoCredential, error) {
	return s.credentials.List()
}

func (s *RepoCredentialService) ListByUser(userID int64) ([]model.RepoCredential, error) {
	return s.credentials.ListByUser(userID)
}

func (s *RepoCredentialService) Get(id int64) (*model.RepoCredential, error) {
	return s.credentials.GetByID(id)
}

type CredentialCreateInput struct {
	Name      string
	Username  string
	Password  string
	CreatedBy int64
}

func (s *RepoCredentialService) Create(input CredentialCreateInput) (*model.RepoCredential, error) {
	cred := &model.RepoCredential{
		Name:      input.Name,
		Username:  input.Username,
		Password:  input.Password,
		CreatedBy: input.CreatedBy,
	}
	if err := s.credentials.Create(cred); err != nil {
		return nil, err
	}
	return cred, nil
}

func (s *RepoCredentialService) Update(id int64, input CredentialCreateInput) (*model.RepoCredential, error) {
	cred, err := s.credentials.GetByID(id)
	if err != nil {
		return nil, err
	}
	cred.Name = input.Name
	cred.Username = input.Username
	if input.Password != "" {
		cred.Password = input.Password
	}
	if err := s.credentials.Update(cred); err != nil {
		return nil, err
	}
	return cred, nil
}

// Delete 删除凭据，有项目引用时拒绝删除
func (s *RepoCredentialService) Delete(id int64) error {
	count, err := s.credentials.CountProjectsByCredential(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该凭据被 %d 个项目引用，无法删除", count)
	}
	return s.credentials.Delete(id)
}

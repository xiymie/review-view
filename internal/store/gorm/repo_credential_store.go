package gormstore

import (
	"review-view/internal/model"
	"gorm.io/gorm"
)

type RepoCredentialStore struct {
	db *gorm.DB
}

func (s *RepoCredentialStore) Create(cred *model.RepoCredential) error {
	return s.db.Create(cred).Error
}

func (s *RepoCredentialStore) Update(cred *model.RepoCredential) error {
	return s.db.Save(cred).Error
}

func (s *RepoCredentialStore) Delete(id int64) error {
	return s.db.Delete(&model.RepoCredential{}, id).Error
}

func (s *RepoCredentialStore) GetByID(id int64) (*model.RepoCredential, error) {
	var cred model.RepoCredential
	if err := s.db.First(&cred, id).Error; err != nil {
		return nil, err
	}
	return &cred, nil
}

func (s *RepoCredentialStore) List() ([]model.RepoCredential, error) {
	var creds []model.RepoCredential
	if err := s.db.Order("id asc").Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

func (s *RepoCredentialStore) ListByUser(userID int64) ([]model.RepoCredential, error) {
	var creds []model.RepoCredential
	if err := s.db.Where("created_by = ?", userID).Order("id asc").Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

func (s *RepoCredentialStore) CountProjectsByCredential(credentialID int64) (int64, error) {
	var count int64
	err := s.db.Model(&model.Project{}).Where("repo_credential_id = ?", credentialID).Count(&count).Error
	return count, err
}

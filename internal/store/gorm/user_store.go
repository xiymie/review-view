package gormstore

import (
	"review-view/internal/model"

	"gorm.io/gorm"
)

type UserStore struct {
	db *gorm.DB
}

func (s *UserStore) Create(u *model.User) error {
	return s.db.Create(u).Error
}

func (s *UserStore) Update(u *model.User) error {
	return s.db.Save(u).Error
}

func (s *UserStore) Delete(id int64) error {
	return s.db.Delete(&model.User{}, id).Error
}

func (s *UserStore) GetByID(id int64) (*model.User, error) {
	var u model.User
	err := s.db.First(&u, id).Error
	return &u, err
}

func (s *UserStore) GetByUsername(username string) (*model.User, error) {
	var u model.User
	err := s.db.Where("username = ?", username).First(&u).Error
	return &u, err
}

func (s *UserStore) List() ([]model.User, error) {
	var users []model.User
	err := s.db.Order("id asc").Find(&users).Error
	return users, err
}

func (s *UserStore) Count() (int64, error) {
	var count int64
	err := s.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

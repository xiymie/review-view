package gormstore

import (
	"errors"
	"time"

	"review-view/internal/model"
	"gorm.io/gorm"
)

type TaskStore struct {
	db *gorm.DB
}

func (s *TaskStore) Create(task *model.Task) error {
	return s.db.Create(task).Error
}

func (s *TaskStore) Update(task *model.Task) error {
	return s.db.Save(task).Error
}

func (s *TaskStore) Delete(id int64) error {
	return s.db.Where("id = ?", id).Delete(&model.Task{}).Error
}

func (s *TaskStore) DeleteByProject(projectID int64) error {
	return s.db.Where("project_id = ?", projectID).Delete(&model.Task{}).Error
}

func (s *TaskStore) GetByID(id int64) (*model.Task, error) {
	var task model.Task
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskStore) ListPending(limit int) ([]model.Task, error) {
	var tasks []model.Task
	query := s.db.Where("status = ?", model.TaskStatusPending).Order("created_at asc, id asc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskStore) ListRecent(limit int) ([]model.Task, error) {
	var tasks []model.Task
	query := s.db.Order("created_at desc, id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskStore) ListByProject(projectID int64, limit int) ([]model.Task, error) {
	var tasks []model.Task
	query := s.db.Where("project_id = ?", projectID).Order("created_at desc, id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskStore) ListByProjectIDs(projectIDs []int64, limit int) ([]model.Task, error) {
	if len(projectIDs) == 0 {
		return nil, nil
	}
	var tasks []model.Task
	query := s.db.Where("project_id IN ?", projectIDs).Order("created_at desc, id desc")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

func (s *TaskStore) FindCompletedRange(projectID int64, fromCommit, toCommit string) (*model.Task, error) {
	var task model.Task
	err := s.db.Where(
		"project_id = ? AND status = ? AND from_commit = ? AND to_commit = ?",
		projectID,
		model.TaskStatusCompleted,
		fromCommit,
		toCommit,
	).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// FindActiveRange 查找同一项目相同 commit 范围的 pending/running 任务，用于触发去重
func (s *TaskStore) FindActiveRange(projectID int64, fromCommit, toCommit string) (*model.Task, error) {
	var task model.Task
	err := s.db.Where(
		"project_id = ? AND status IN ? AND from_commit = ? AND to_commit = ?",
		projectID,
		[]model.TaskStatus{model.TaskStatusPending, model.TaskStatusRunning},
		fromCommit,
		toCommit,
	).First(&task).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskStore) CountRunningByProject(projectID int64) (int64, error) {
	var count int64
	err := s.db.Model(&model.Task{}).
		Where("project_id = ? AND status = ?", projectID, model.TaskStatusRunning).
		Count(&count).Error
	return count, err
}

// ClaimPending 原子地将任务从 pending 状态抢占为 running，返回是否抢占成功
func (s *TaskStore) ClaimPending(taskID int64) bool {
	result := s.db.Model(&model.Task{}).
		Where("id = ? AND status = ?", taskID, model.TaskStatusPending).
		Update("status", model.TaskStatusRunning)
	return result.RowsAffected == 1
}

func (s *TaskStore) AppendLog(taskID int64, level model.TaskLogLevel, message string) error {
	return s.db.Create(&model.TaskLog{
		TaskID:  taskID,
		Level:   level,
		Message: message,
	}).Error
}

func (s *TaskStore) AppendLogs(logs []model.TaskLog) error {
	if len(logs) == 0 {
		return nil
	}
	return s.db.Create(&logs).Error
}

// UpdateResult 只更新 task 的 result 字段，避免 Save 全量写入的竞态问题
func (s *TaskStore) UpdateResult(taskID int64, result string) error {
	return s.db.Model(&model.Task{}).Where("id = ?", taskID).Update("result", result).Error
}

func (s *TaskStore) ListLogs(taskID int64) ([]model.TaskLog, error) {
	var logs []model.TaskLog
	if err := s.db.Where("task_id = ?", taskID).Order("id asc").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// RecoverRunning 将所有 running 状态的任务批量标记为 failed，返回受影响的行数
func (s *TaskStore) RecoverRunning() (int64, error) {
	now := time.Now()
	result := s.db.Model(&model.Task{}).
		Where("status = ?", model.TaskStatusRunning).
		Updates(map[string]interface{}{
			"status":       model.TaskStatusFailed,
			"error_message": "服务重启，任务被强制终止",
			"finished_at":  now,
		})
	return result.RowsAffected, result.Error
}

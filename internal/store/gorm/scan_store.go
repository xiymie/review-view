package gormstore

import (
	"time"

	"review-view/internal/model"
	"gorm.io/gorm"
)

// ScanScheduleStore

type ScanScheduleStore struct {
	db *gorm.DB
}

func (s *ScanScheduleStore) Create(v *model.ScanSchedule) error {
	return s.db.Create(v).Error
}

func (s *ScanScheduleStore) Update(v *model.ScanSchedule) error {
	return s.db.Save(v).Error
}

func (s *ScanScheduleStore) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 查出所有 job id
		var jobIDs []int64
		tx.Model(&model.ScanJob{}).Where("schedule_id = ?", id).Pluck("id", &jobIDs)
		// 级联删 branch_results
		if len(jobIDs) > 0 {
			if err := tx.Where("job_id IN ?", jobIDs).Delete(&model.ScanBranchResult{}).Error; err != nil {
				return err
			}
		}
		// 删 jobs
		if err := tx.Where("schedule_id = ?", id).Delete(&model.ScanJob{}).Error; err != nil {
			return err
		}
		// 删 schedule
		return tx.Delete(&model.ScanSchedule{}, id).Error
	})
}

func (s *ScanScheduleStore) GetByID(id int64) (*model.ScanSchedule, error) {
	var v model.ScanSchedule
	if err := s.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *ScanScheduleStore) List() ([]model.ScanSchedule, error) {
	var list []model.ScanSchedule
	if err := s.db.Order("id asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ScanScheduleStore) ListEnabled() ([]*model.ScanSchedule, error) {
	var list []*model.ScanSchedule
	if err := s.db.Where("enabled = ?", true).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ScanJobStore

type ScanJobStore struct {
	db *gorm.DB
}

func (s *ScanJobStore) Create(v *model.ScanJob) error {
	return s.db.Create(v).Error
}

func (s *ScanJobStore) Update(v *model.ScanJob) error {
	return s.db.Save(v).Error
}

func (s *ScanJobStore) Delete(id int64) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("job_id = ?", id).Delete(&model.ScanBranchResult{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.ScanJob{}, id).Error
	})
}

func (s *ScanJobStore) GetByID(id int64) (*model.ScanJob, error) {
	var v model.ScanJob
	if err := s.db.First(&v, id).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

func (s *ScanJobStore) ListBySchedule(scheduleID int64, limit int) ([]model.ScanJob, error) {
	var list []model.ScanJob
	q := s.db.Where("schedule_id = ?", scheduleID).Order("id desc")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ScanJobStore) ListBranchResults(jobID int64) ([]model.ScanBranchResult, error) {
	var list []model.ScanBranchResult
	if err := s.db.Where("job_id = ?", jobID).Order("branch_name asc").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (s *ScanJobStore) CreateBranchResult(v *model.ScanBranchResult) error {
	return s.db.Create(v).Error
}

// ListJobsOlderThan 返回 triggered_at 早于 before 且有 report_path 的 job 列表
func (s *ScanJobStore) ListJobsOlderThan(scheduleID int64, before time.Time) ([]model.ScanJob, error) {
	var list []model.ScanJob
	err := s.db.Where("schedule_id = ? AND triggered_at < ? AND report_path != ''", scheduleID, before).
		Find(&list).Error
	return list, err
}

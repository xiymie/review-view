package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"review-view/internal/model"
	"review-view/internal/store"
)

// ScanScheduler 每分钟检查一次所有启用的巡检配置，到达设定时间就触发
type ScanScheduler struct {
	svc      *ScanService
	schedules store.ScanScheduleStore
	settings *SettingsService
}

func NewScanScheduler(svc *ScanService, schedules store.ScanScheduleStore, settings *SettingsService) *ScanScheduler {
	return &ScanScheduler{svc: svc, schedules: schedules, settings: settings}
}

func (s *ScanScheduler) Loop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(ctx, now)
		}
	}
}

func (s *ScanScheduler) tick(ctx context.Context, now time.Time) {
	list, err := s.schedules.ListEnabled()
	if err != nil {
		log.Printf("[ScanScheduler] list enabled: %v", err)
		return
	}

	// 全局默认时间
	globalTime, _ := s.settings.GetRaw(model.GlobalConfigKeyScanTime)
	if globalTime == "" {
		globalTime = "09:00"
	}

	currentHM := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

	for _, sched := range list {
		target := sched.ScanTime
		if target == "" {
			target = globalTime
		}
		if matchTime(currentHM, target) {
			go func(id int64) {
				if err := s.svc.RunSchedule(ctx, id); err != nil {
					log.Printf("[ScanScheduler] run schedule %d: %v", id, err)
				}
			}(sched.ID)
		}
	}
}

// matchTime 精确匹配 HH:MM
func matchTime(current, target string) bool {
	return strings.TrimSpace(current) == strings.TrimSpace(target)
}

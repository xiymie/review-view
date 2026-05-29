package service

import (
	"strconv"

	"review-view/internal/model"
	"review-view/internal/store"
)

type Settings struct {
	MaxConcurrentTasks       int
	OverflowStrategy         model.OverflowStrategy
	TaskTimeout              int
	RepoBaseDir              string
	ScheduledScanUnchanged   bool
	ManualScanUnchanged      bool
}

type SettingsInput struct {
	MaxConcurrentTasks       int
	OverflowStrategy         model.OverflowStrategy
	TaskTimeout              int
	RepoBaseDir              string
	ScheduledScanUnchanged   bool
	ManualScanUnchanged      bool
}

type SettingsService struct {
	configs store.GlobalConfigStore
}

func NewSettingsService(configs store.GlobalConfigStore) *SettingsService {
	return &SettingsService{configs: configs}
}

func (s *SettingsService) Get() (*Settings, error) {
	if err := s.configs.EnsureDefaults(); err != nil {
		return nil, err
	}

	values, err := s.configs.List()
	if err != nil {
		return nil, err
	}

	settings := &Settings{
		MaxConcurrentTasks:     3,
		OverflowStrategy:       model.OverflowStrategyQueue,
		TaskTimeout:            30,
		RepoBaseDir:            "./repos",
		ScheduledScanUnchanged: false,
		ManualScanUnchanged:    true,
	}
	for _, item := range values {
		switch item.Key {
		case model.GlobalConfigKeyMaxConcurrentTasks:
			if v, err := strconv.Atoi(item.Value); err == nil {
				settings.MaxConcurrentTasks = v
			}
		case model.GlobalConfigKeyOverflowStrategy:
			settings.OverflowStrategy = model.OverflowStrategy(item.Value)
		case model.GlobalConfigKeyTaskTimeout:
			if v, err := strconv.Atoi(item.Value); err == nil {
				settings.TaskTimeout = v
			}
		case model.GlobalConfigKeyRepoBaseDir:
			settings.RepoBaseDir = item.Value
		case model.GlobalConfigKeyScheduledScanUnchanged:
			settings.ScheduledScanUnchanged = item.Value == "true"
		case model.GlobalConfigKeyManualScanUnchanged:
			settings.ManualScanUnchanged = item.Value == "true"
		}
	}
	return settings, nil
}

func (s *SettingsService) Update(input SettingsInput) error {
	if err := s.configs.Set(model.GlobalConfigKeyMaxConcurrentTasks, strconv.Itoa(input.MaxConcurrentTasks)); err != nil {
		return err
	}
	if err := s.configs.Set(model.GlobalConfigKeyOverflowStrategy, string(input.OverflowStrategy)); err != nil {
		return err
	}
	if err := s.configs.Set(model.GlobalConfigKeyTaskTimeout, strconv.Itoa(input.TaskTimeout)); err != nil {
		return err
	}
	if err := s.configs.Set(model.GlobalConfigKeyRepoBaseDir, input.RepoBaseDir); err != nil {
		return err
	}
	if err := s.configs.Set(model.GlobalConfigKeyScheduledScanUnchanged, strconv.FormatBool(input.ScheduledScanUnchanged)); err != nil {
		return err
	}
	return s.configs.Set(model.GlobalConfigKeyManualScanUnchanged, strconv.FormatBool(input.ManualScanUnchanged))
}

func (s *SettingsService) GetRaw(key string) (string, error) {
	return s.configs.Get(key)
}

func (s *SettingsService) SetSMTP(host, port, username, password, from, fromName, tls string) error {
	pairs := [][2]string{
		{model.GlobalConfigKeySMTPHost, host},
		{model.GlobalConfigKeySMTPPort, port},
		{model.GlobalConfigKeySMTPUsername, username},
		{model.GlobalConfigKeySMTPFrom, from},
		{model.GlobalConfigKeySMTPFromName, fromName},
		{model.GlobalConfigKeySMTPTLS, tls},
	}
	for _, p := range pairs {
		if err := s.configs.Set(p[0], p[1]); err != nil {
			return err
		}
	}
	// 密码只在非空时更新，避免每次保存清空密码
	if password != "" {
		if err := s.configs.Set(model.GlobalConfigKeySMTPPassword, password); err != nil {
			return err
		}
	}
	return nil
}

// GetSMTPConfig 供 notifier 读取 SMTP 配置
func (s *SettingsService) GetSMTPConfig() (host, port, username, password, from, fromName, tls string) {
	host, _ = s.configs.Get(model.GlobalConfigKeySMTPHost)
	port, _ = s.configs.Get(model.GlobalConfigKeySMTPPort)
	username, _ = s.configs.Get(model.GlobalConfigKeySMTPUsername)
	password, _ = s.configs.Get(model.GlobalConfigKeySMTPPassword)
	from, _ = s.configs.Get(model.GlobalConfigKeySMTPFrom)
	fromName, _ = s.configs.Get(model.GlobalConfigKeySMTPFromName)
	tls, _ = s.configs.Get(model.GlobalConfigKeySMTPTLS)
	return
}

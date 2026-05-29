package service

import (
	"encoding/json"
	"fmt"

	"review-view/internal/model"
	"review-view/internal/store"
)

type ModelConfigCreateInput struct {
	Name           string
	Type           model.ModelType
	BaseURL        string
	APIKey         string
	Model          string
	Prompt         string
	MaxContext     int
	EnableThinking bool
	CLIPath        string
	EnvVarsJSON    string
	MaxTurns       *int
}

type ModelConfigService struct {
	store store.ModelConfigStore
}

func NewModelConfigService(store store.ModelConfigStore) *ModelConfigService {
	return &ModelConfigService{store: store}
}

// BuildConfig 根据输入构建 ModelConfig 对象，不持久化，用于测试连接
func (s *ModelConfigService) BuildConfig(input ModelConfigCreateInput) (*model.ModelConfig, error) {
	config := &model.ModelConfig{
		Name:           input.Name,
		Type:           input.Type,
		BaseURL:        input.BaseURL,
		APIKey:         input.APIKey,
		Model:          input.Model,
		Prompt:         input.Prompt,
		MaxContext:     input.MaxContext,
		EnableThinking: input.EnableThinking,
	}

	if input.Type == model.ModelTypeClaudeCLI {
		envVars := map[string]string{}
		if input.EnvVarsJSON != "" {
			if err := json.Unmarshal([]byte(input.EnvVarsJSON), &envVars); err != nil {
				return nil, err
			}
		}
		extra := model.ClaudeCLIExtraConfig{
			CLIPath: input.CLIPath,
			EnvVars: envVars,
		}
		if input.MaxTurns != nil {
			extra.MaxTurns = *input.MaxTurns
		}
		payload, err := json.Marshal(extra)
		if err != nil {
			return nil, err
		}
		config.ExtraConfig = string(payload)
	}

	return config, nil
}

func (s *ModelConfigService) Delete(id int64) error {
	count, err := s.store.CountByModelConfig(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("该模型配置被 %d 个项目使用，请先更换项目的模型配置后再删除", count)
	}
	return s.store.Delete(id)
}

func (s *ModelConfigService) List() ([]model.ModelConfig, error) {
	return s.store.List()
}

func (s *ModelConfigService) Get(id int64) (*model.ModelConfig, error) {
	return s.store.GetByID(id)
}

func (s *ModelConfigService) Create(input ModelConfigCreateInput) (*model.ModelConfig, error) {
	config := &model.ModelConfig{
		Name:           input.Name,
		Type:           input.Type,
		BaseURL:        input.BaseURL,
		APIKey:         input.APIKey,
		Model:          input.Model,
		Prompt:         input.Prompt,
		MaxContext:     input.MaxContext,
		EnableThinking: input.EnableThinking,
	}

	if input.Type == model.ModelTypeClaudeCLI {
		envVars := map[string]string{}
		if input.EnvVarsJSON != "" {
			if err := json.Unmarshal([]byte(input.EnvVarsJSON), &envVars); err != nil {
				return nil, err
			}
		}
		extra := model.ClaudeCLIExtraConfig{
			CLIPath: input.CLIPath,
			EnvVars: envVars,
		}
		if input.MaxTurns != nil {
			extra.MaxTurns = *input.MaxTurns
		}
		payload, err := json.Marshal(extra)
		if err != nil {
			return nil, err
		}
		config.ExtraConfig = string(payload)
	}

	if err := s.store.Create(config); err != nil {
		return nil, err
	}
	return config, nil
}

func (s *ModelConfigService) Update(id int64, input ModelConfigCreateInput) (*model.ModelConfig, error) {
	config, err := s.store.GetByID(id)
	if err != nil {
		return nil, err
	}

	config.Name = input.Name
	config.Type = input.Type
	config.BaseURL = input.BaseURL
	config.APIKey = input.APIKey
	config.Model = input.Model
	config.Prompt = input.Prompt
	config.MaxContext = input.MaxContext
	config.EnableThinking = input.EnableThinking

	if input.Type == model.ModelTypeClaudeCLI {
		envVars := map[string]string{}
		if input.EnvVarsJSON != "" {
			if err := json.Unmarshal([]byte(input.EnvVarsJSON), &envVars); err != nil {
				return nil, err
			}
		}
		extra := model.ClaudeCLIExtraConfig{
			CLIPath: input.CLIPath,
			EnvVars: envVars,
		}
		if input.MaxTurns != nil {
			extra.MaxTurns = *input.MaxTurns
		}
		payload, err := json.Marshal(extra)
		if err != nil {
			return nil, err
		}
		config.ExtraConfig = string(payload)
	} else {
		config.ExtraConfig = ""
	}

	if err := s.store.Update(config); err != nil {
		return nil, err
	}
	return config, nil
}

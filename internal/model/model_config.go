package model

import (
	"encoding/json"
	"time"
)

type ModelType string

const (
	ModelTypeOpenAI    ModelType = "openai"
	ModelTypeAnthropic ModelType = "anthropic"
	ModelTypeOllama    ModelType = "ollama"
	ModelTypeDeepSeek  ModelType = "deepseek"
	ModelTypeGemini    ModelType = "gemini"
	ModelTypeMistral   ModelType = "mistral"
	ModelTypeClaudeCLI ModelType = "claude_cli"
)

type ModelConfig struct {
	ID             int64     `gorm:"primaryKey"`
	Name           string    `gorm:"not null"`
	Type           ModelType `gorm:"not null;index"`
	BaseURL        string
	APIKey         string
	Model          string
	Prompt         string `gorm:"not null"`
	MaxContext     int
	EnableThinking bool
	ExtraConfig    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ClaudeCLIExtraConfig struct {
	CLIPath  string            `json:"cli_path"`
	EnvVars  map[string]string `json:"env_vars"`
	MaxTurns int               `json:"max_turns"`
}

func (m ModelConfig) DecodeExtraConfig(out any) error {
	if m.ExtraConfig == "" {
		return nil
	}
	return json.Unmarshal([]byte(m.ExtraConfig), out)
}

package review

import (
	"fmt"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/config"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
	"github.com/mozilla-ai/any-llm-go/providers/deepseek"
	"github.com/mozilla-ai/any-llm-go/providers/gemini"
	"github.com/mozilla-ai/any-llm-go/providers/mistral"
	"github.com/mozilla-ai/any-llm-go/providers/ollama"
	"github.com/mozilla-ai/any-llm-go/providers/openai"
	"review-view/internal/model"
)

type ProviderFactory func(config *model.ModelConfig) (CompletionProvider, error)

func NewProvider(configModel *model.ModelConfig) (CompletionProvider, error) {
	opts := []config.Option{}
	if configModel.APIKey != "" {
		opts = append(opts, anyllm.WithAPIKey(configModel.APIKey))
	}
	if configModel.BaseURL != "" {
		opts = append(opts, anyllm.WithBaseURL(configModel.BaseURL))
	}

	switch configModel.Type {
	case model.ModelTypeOpenAI:
		return openai.New(opts...)
	case model.ModelTypeAnthropic:
		return anthropic.New(opts...)
	case model.ModelTypeOllama:
		return ollama.New(opts...)
	case model.ModelTypeDeepSeek:
		return deepseek.New(opts...)
	case model.ModelTypeGemini:
		return gemini.New(opts...)
	case model.ModelTypeMistral:
		return mistral.New(opts...)
	default:
		return nil, fmt.Errorf("unsupported llm provider type %q", configModel.Type)
	}
}

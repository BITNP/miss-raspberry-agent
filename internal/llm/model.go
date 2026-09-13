// Package llm owns the LLM inference concerns: constructing the OpenAI-compatible
// chat model and adapting it to strict providers.
package llm

import (
	"context"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"

	"miss-raspberry-agent/internal/config"
)

// NewChatModel builds an OpenAI-compatible chat model from the given configuration
// without forcing a specific output format.
func NewChatModel(ctx context.Context, cfg config.ModelConfig) (*einoopenai.ChatModel, error) {
	return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Name,
	})
}

// NewJSONModel builds an OpenAI-compatible chat model that enforces a JSON-object
// response format so agent output stays parseable.
func NewJSONModel(ctx context.Context, cfg config.ModelConfig) (*einoopenai.ChatModel, error) {
	return einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
		Model:   cfg.Name,
		ResponseFormat: &einoopenai.ChatCompletionResponseFormat{
			Type: einoopenai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
}

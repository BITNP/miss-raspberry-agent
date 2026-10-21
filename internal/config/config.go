// Package config loads service configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	envModelAPIKey  = "MODEL_API_KEY"
	envModelBaseURL = "MODEL_BASE_URL"
	envModelName    = "MODEL_NAME"

	envNapcatWebSocketURL = "NAPCAT_WS_URL"
	envNapcatAccessToken  = "NAPCAT_ACCESS_TOKEN"

	envHTTPAddr     = "HTTP_ADDR"
	envHTTPAPIToken = "API_TOKEN"

	envSystemPrompt = "SYSTEM_PROMPT"

	defaultBaseURL            = "https://api.openai.com/v1"
	defaultModelStr           = "gpt-4o-mini"
	defaultNapcatWebSocketURL = "ws://127.0.0.1:3001"
	defaultHTTPAddr           = ":4514"
)

// Config is the top-level application configuration.
type Config struct {
	Model  ModelConfig
	Napcat NapcatConfig
	HTTP   HTTPConfig
	Agent  AgentConfig
}

// AgentConfig describes the externally callable agents.
type AgentConfig struct {
	// SystemPrompt overrides the main agent's built-in persona when non-empty.
	// The built-in default is documented in internal/agent/main.
	SystemPrompt string
}

// ModelConfig describes which LLM endpoint to call.
type ModelConfig struct {
	APIKey  string
	BaseURL string
	Name    string
}

// NapcatConfig describes the NapCat WebSocket connection.
type NapcatConfig struct {
	WebSocketURL string
	AccessToken  string
}

// HTTPConfig describes the inbound HTTP API server.
type HTTPConfig struct {
	Addr     string
	APIToken string
}

// Load reads configuration from the process environment.
// Startup fails if a required variable is missing.
func Load() (Config, error) {
	cfg := ModelConfig{
		APIKey: os.Getenv(envModelAPIKey),
	}
	if cfg.APIKey == "" {
		return Config{}, errors.New("environment variable MODEL_API_KEY is required")
	}

	cfg.BaseURL = envOrDefault(envModelBaseURL, defaultBaseURL)
	cfg.Name = envOrDefault(envModelName, defaultModelStr)

	apiToken := os.Getenv(envHTTPAPIToken)
	if apiToken == "" {
		return Config{}, errors.New("environment variable API_TOKEN is required")
	}

	// adk formats the agent instruction like an f-string when session values are
	// present, so braces in the prompt would be interpreted instead of sent
	// literally. Reject them at startup rather than failing mid-conversation.
	systemPrompt := os.Getenv(envSystemPrompt)
	if strings.ContainsAny(systemPrompt, "{}") {
		return Config{}, fmt.Errorf("environment variable %s must not contain curly braces", envSystemPrompt)
	}

	return Config{
		Model: cfg,
		Napcat: NapcatConfig{
			WebSocketURL: envOrDefault(envNapcatWebSocketURL, defaultNapcatWebSocketURL),
			AccessToken:  os.Getenv(envNapcatAccessToken),
		},
		HTTP: HTTPConfig{
			Addr:     envOrDefault(envHTTPAddr, defaultHTTPAddr),
			APIToken: apiToken,
		},
		Agent: AgentConfig{
			SystemPrompt: systemPrompt,
		},
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

package llm

import (
	"net/http"

	"github.com/roman-samoilenko/sreagent/config"

	"github.com/tmc/langchaingo/llms/openai"
)

func NewOpenRouter(cfg *config.LLMConfig) (*openai.LLM, error) {
	model := cfg.OpenRouterCurrentModel
	if model == "" {
		model = "openrouter/auto:free"
	}

	customClient := &http.Client{}

	return openai.New(
		openai.WithToken(cfg.OpenRouterAPIKey),
		openai.WithModel(model),
		openai.WithBaseURL("https://openrouter.ai/api/v1"),
		openai.WithHTTPClient(customClient),
	)
}

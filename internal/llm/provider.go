package llm

import (
	"fmt"

	"github.com/roman-samoilenko/sreagent/config"

	"github.com/tmc/langchaingo/llms"
)

func NewLLM(llmcfg *config.LLMConfig) (llms.Model, error) {
	switch llmcfg.Provider {
	case "openrouter":
		return NewOpenRouter(llmcfg)

	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", llmcfg.Provider)
	}
}

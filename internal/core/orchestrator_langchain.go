package core

import (
	"context"
	"fmt"
	"strings"

	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
	"github.com/tmc/langchaingo/agents"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/chains"
)

type Orchestrator interface {
	RunTask(ctx context.Context, taskID, query string) (string, error)
}

type LangchainOrchestrator struct {
	llm    llms.Model
	agent  *agents.Executor
	memory schema.Memory
}

func NewLangchainOrchestrator(
	llm llms.Model,
	allTools []tools.Tool,
	cfg *config.Config,
) (*LangchainOrchestrator, error) {

	// Краткосрочная память – ConversationBuffer (без лимита токенов, проще и надёжнее)
	mem := memory.NewConversationBuffer()

	// Системный промпт с перечнем репозиториев и строгим форматом
	repoList := "Monitored repositories:\n- " + strings.Join(cfg.Repositories, "\n- ")
	prefix := cfg.Agents.Prompt + "\n\n" + repoList + `

You have access to the following tools:

{{.tool_descriptions}}

You MUST respond using the ReAct format:
Thought: <your reasoning>
Action: <tool name>
Action Input: <JSON>
Observation: <tool result>
... (repeat if needed) ...
Final Answer: <plain text answer to user>

If you don't need a tool, output:
Final Answer: <your message>`

	agent := agents.NewConversationalAgent(
		llm,
		allTools,
		agents.WithPromptPrefix(prefix),
	)

	executor := agents.NewExecutor(
		agent,
		agents.WithMaxIterations(cfg.MaxReActSteps),
		agents.WithMemory(mem),
	)

	return &LangchainOrchestrator{
		llm:    llm,
		agent:  executor,
		memory: mem,
	}, nil
}

func (o *LangchainOrchestrator) RunTask(ctx context.Context, taskID, query string) (string, error) {
	result, err := chains.Call(ctx, o.agent, map[string]any{"input": query})

	// Логируем состояние краткосрочной памяти
	if memVars, loadErr := o.memory.LoadMemoryVariables(ctx, map[string]any{}); loadErr == nil {
		if history, ok := memVars["history"]; ok {
			if messages, ok := history.([]llms.ChatMessage); ok {
				logger.Debug("Conversation memory size", "messages", len(messages))
			} else {
				logger.Debug("Conversation memory content", "history", fmt.Sprintf("%v", history))
			}
		}
	}

	if err != nil {
		if extracted := extractFinalAnswer(err.Error()); extracted != "" {
			return extracted, nil
		}
		return "", fmt.Errorf("agent call failed: %w", err)
	}

	output, ok := result["output"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected output type")
	}
	return output, nil
}

func extractFinalAnswer(errStr string) string {
	const marker = "Final Answer:"
	idx := strings.LastIndex(errStr, marker)
	if idx == -1 {
		return ""
	}
	raw := strings.TrimSpace(errStr[idx+len(marker):])
	raw = strings.TrimRight(raw, `"`)
	return strings.TrimSpace(raw)
}

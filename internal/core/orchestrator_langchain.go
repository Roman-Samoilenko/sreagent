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
	)

	return &LangchainOrchestrator{
		llm:    llm,
		agent:  executor,
		memory: mem,
	}, nil
}

func (o *LangchainOrchestrator) RunTask(ctx context.Context, taskID, query string) (string, error) {
	result, err := o.agent.Call(ctx, map[string]any{"input": query})

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

// extractFinalAnswer pulls the text after the last "Final Answer:" marker
// out of an agent error message. langchaingo's ReAct parser sometimes
// returns the final answer wrapped in an error when it doesn't match the
// expected intermediate-step format exactly; this recovers it instead of
// discarding a valid answer.
func extractFinalAnswer(errStr string) string {
	const marker = "Final Answer:"
	idx := strings.LastIndex(errStr, marker)
	if idx == -1 {
		return ""
	}
	raw := strings.TrimSpace(errStr[idx+len(marker):])
	raw = strings.Trim(raw, `"`)
	return strings.TrimSpace(raw)
}

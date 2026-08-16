package core

import (
	"context"
	"errors"
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
	mem := memory.NewConversationBuffer()

	repoList := "Monitored repositories:\n- " + strings.Join(cfg.Repositories, "\n- ")
	prefix := cfg.Agents.Prompt + "\n\n" + repoList + `

You have access to the following tools:

{{.tool_descriptions}}

STRICT OUTPUT FORMAT. Every single response you produce MUST be one of the
following two shapes, with no other text before or after:

Thought: <your reasoning, one or two sentences>
Action: <the exact tool name>
Action Input: <a single valid JSON object, no markdown fences, no comments>

OR, when you have the final answer and need no more tools:

Thought: <your reasoning, one or two sentences>
Final Answer: <plain text answer to the user>

Rules:
- Never omit the "Thought:" line.
- Never write anything after "Final Answer:" other than the answer itself.
- Never wrap Action Input in markdown code fences (no triple backticks).
- Never call a tool that is not in the tool list above.
- If you are not calling a tool, you MUST use "Final Answer:", not plain prose.`

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
		logger.Error("agent call failed", "task_id", taskID, "raw_error", err.Error())

		if extracted := extractFinalAnswer(err.Error()); extracted != "" {
			return extracted, nil
		}

		if isParseError(err) {
			return "", fmt.Errorf(
				"the model's response didn't follow the required Thought/Action or Final Answer format (see logs for the raw output); this usually means the current LLM isn't reliable enough for ReAct tool use, try a stronger model: %w",
				err,
			)
		}

		return "", fmt.Errorf("agent call failed: %w", err)
	}

	output, ok := result["output"].(string)
	if !ok {
		return "", errors.New("unexpected output type")
	}
	return output, nil
}

func isParseError(err error) bool {
	return strings.Contains(err.Error(), "unable to parse agent output")
}

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

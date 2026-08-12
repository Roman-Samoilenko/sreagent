package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/tools"
)

// LangchainTool adapts one MCP tool (from any connected server) to the
// langchaingo tools.Tool interface, so the ReAct agent can call it exactly
// like any other tool.
type LangchainTool struct {
	manager  *Manager
	fullName string
	desc     string
}

var _ tools.Tool = (*LangchainTool)(nil)

func newLangchainTool(manager *Manager, handle *ToolHandle) *LangchainTool {
	desc := handle.Tool.Description
	if desc == "" {
		desc = fmt.Sprintf("Tool %s from MCP server %s", handle.ToolName, handle.Server)
	}
	desc += "\nInput MUST be a single JSON object with the arguments described above."

	return &LangchainTool{
		manager:  manager,
		fullName: handle.FullName(),
		desc:     desc,
	}
}

func (t *LangchainTool) Name() string        { return t.fullName }
func (t *LangchainTool) Description() string { return t.desc }

func (t *LangchainTool) Call(ctx context.Context, input string) (string, error) {
	cleaned := stripMarkdownFence(strings.TrimSpace(input))

	arguments := map[string]any{}
	if cleaned != "" {
		if err := json.Unmarshal([]byte(cleaned), &arguments); err != nil {
			return fmt.Sprintf("ERROR: input must be a valid JSON object, got %q: %v", input, err), nil
		}
	}

	result, err := t.manager.CallTool(ctx, t.fullName, arguments)
	if err != nil {
		return "", fmt.Errorf("tool %s error: %w", t.fullName, err)
	}
	return result, nil
}

func stripMarkdownFence(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

// BuildTools connects the langchain agent's tool list to every tool
// currently known to the manager. Call LoadTools first to (re)populate the
// catalog.
func BuildTools(manager *Manager) []tools.Tool {
	var result []tools.Tool
	for _, name := range manager.ToolNames() {
		handle, err := manager.ToolHandle(name)
		if err != nil {
			continue
		}
		result = append(result, newLangchainTool(manager, handle))
	}
	return result
}

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/tools"
)

// MCPTool оборачивает инструмент MCP для использования в LangChainGo
type MCPTool struct {
	fullName    string // "server_tool"
	name        string // "server_tool" (для langchaingo)
	description string
	manager     *Manager
}

// Проверка интерфейса
var _ tools.Tool = (*MCPTool)(nil)

// NewMCPTool создаёт новый инструмент
func NewMCPTool(manager *Manager, toolHandle *ToolHandle) *MCPTool {
	fullName := fmt.Sprintf("%s_%s", toolHandle.Server, toolHandle.Metadata.Name)

	// Улучшенное описание для агента
	desc := toolHandle.Metadata.Description
	if desc == "" {
		desc = fmt.Sprintf("Tool %s from MCP server %s", toolHandle.Metadata.Name, toolHandle.Server)
	}

	return &MCPTool{
		fullName:    fullName,
		name:        fullName,
		description: desc,
		manager:     manager,
	}
}

// Name возвращает имя инструмента
func (t *MCPTool) Name() string {
	return t.name
}

// Description возвращает описание инструмента
func (t *MCPTool) Description() string {
	return t.description
}

// Call выполняет инструмент
func (t *MCPTool) Call(ctx context.Context, input string) (string, error) {
	cleaned := stripMarkdownFence(strings.TrimSpace(input))

	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &arguments); err != nil {
		return fmt.Sprintf("ERROR:invalid JSON input, must be a valid JSON object, got: %q (%v)", input, err), nil
	}

	result, err := t.manager.CallTool(ctx, t.fullName, arguments)
	if err != nil {
		return "", fmt.Errorf("tool %s error: %w", t.fullName, err)
	}
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}

func stripMarkdownFence(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")

	return strings.TrimSpace(s)
}

// MCPToolFactory создаёт набор инструментов из MCP Manager
type MCPToolFactory struct {
	manager *Manager
}

// NewMCPToolFactory создаёт новую фабрику инструментов
func NewMCPToolFactory(manager *Manager) *MCPToolFactory {
	return &MCPToolFactory{
		manager: manager,
	}
}

// BuildTools создаёт набор инструментов для агента
func (f *MCPToolFactory) BuildTools(ctx context.Context) ([]tools.Tool, error) {
	// Загружаем инструменты от всех серверов
	if err := f.manager.LoadToolsFromServers(ctx); err != nil {
		return nil, fmt.Errorf("failed to load tools from servers: %w", err)
	}

	toolNames := f.manager.GetToolNames(ctx)
	var result []tools.Tool

	for _, toolFullName := range toolNames {
		handle, err := f.manager.GetToolMetadata(toolFullName)
		if err != nil {
			continue // Пропускаем недоступные инструменты
		}

		tool := NewMCPTool(f.manager, handle)
		result = append(result, tool)
	}

	return result, nil
}

// GetToolNames возвращает список имён всех инструментов
func (f *MCPToolFactory) GetToolNames() []string {
	return f.manager.GetToolNames(context.Background())
}

// GetToolDescription возвращает описание инструмента
func (f *MCPToolFactory) GetToolDescription(toolName string) (string, error) {
	handle, err := f.manager.GetToolMetadata(toolName)
	if err != nil {
		return "", err
	}
	return handle.Metadata.Description, nil
}

// ParseToolCall парсит вызов инструмента из текста агента
// Например: "Action: server_tool_name\nAction Input: {...}"
func ParseToolCall(text string) (toolName string, input string, err error) {
	// Ищем "Action: <имя инструмента>"
	actionStart := strings.Index(text, "Action:")
	if actionStart == -1 {
		return "", "", fmt.Errorf("no Action found in text")
	}

	// Ищем начало следующей строки
	actionStart += len("Action:")
	actionEnd := strings.Index(text[actionStart:], "\n")
	if actionEnd == -1 {
		actionEnd = len(text) - actionStart
	}

	toolName = strings.TrimSpace(text[actionStart : actionStart+actionEnd])

	// Ищем "Action Input: <ввод>"
	inputStart := strings.Index(text, "Action Input:")
	if inputStart == -1 {
		return toolName, "", nil
	}

	inputStart += len("Action Input:")
	inputEnd := strings.Index(text[inputStart:], "\n")
	if inputEnd == -1 {
		inputEnd = len(text) - inputStart
	}

	input = strings.TrimSpace(text[inputStart : inputStart+inputEnd])
	return toolName, input, nil
}

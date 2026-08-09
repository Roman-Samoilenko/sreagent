package mcp

import "context"

type ToolCall struct {
	ServerName string
	ToolName   string
	Arguments  map[string]any
}

type ToolResult struct {
	Content []byte
	Error   error
}

type MCPServer interface {
	Name() string
	Initialize(ctx context.Context) error
	HandleToolCall(ctx context.Context, toolName string, args map[string]any) ([]byte, error)
	ListTools() []string
}

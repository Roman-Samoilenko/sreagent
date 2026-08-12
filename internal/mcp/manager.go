package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConn is a single connection to one external MCP server.
type ServerConn struct {
	Name    string
	Session *mcp.ClientSession
}

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolHandle links a tool to the server that hosts it.
type ToolHandle struct {
	Server   string // logical server name, e.g. "github"
	ToolName string // the tool's name exactly as the server defines it
	Tool     *mcp.Tool
}

// FullName returns the name under which the tool is exposed to the agent:
// "<server>_<tool>", e.g. "github_get_repo".
func (h *ToolHandle) FullName() string {
	return h.Server + "_" + h.ToolName
}

// Manager owns connections to zero or more MCP servers and the merged tool
// catalog across all of them.
type Manager struct {
	mu    sync.RWMutex
	conns map[string]*ServerConn
	tools map[string]*ToolHandle // fullName -> handle
}

// NewManager creates an empty Manager.
func NewManager() *Manager {
	return &Manager{
		conns: make(map[string]*ServerConn),
		tools: make(map[string]*ToolHandle),
	}
}

// Connect connects to an MCP server over the given transport and registers
// it under name. name is used purely as a local namespace prefix for tools
// (e.g. "github_get_repo"); it does not need to match the server's own
// self-reported name.
func (m *Manager) Connect(ctx context.Context, name string, transport mcp.Transport) error {
	client := mcp.NewClient(&mcp.Implementation{Name: "sreagent", Version: "0.1.0"}, nil)

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("mcp: connect to %q: %w", name, err)
	}

	m.mu.Lock()
	m.conns[name] = &ServerConn{Name: name, Session: session}
	m.mu.Unlock()

	return nil
}

// LoadTools lists tools from every connected server and merges them into
// the manager's tool catalog under "<server>_<tool>" names. Safe to call
// again after reconnecting a server to refresh the catalog.
func (m *Manager) LoadTools(ctx context.Context) error {
	m.mu.RLock()
	conns := make([]*ServerConn, 0, len(m.conns))
	for _, c := range m.conns {
		conns = append(conns, c)
	}
	m.mu.RUnlock()

	newTools := make(map[string]*ToolHandle)
	for _, conn := range conns {
		for tool, err := range conn.Session.Tools(ctx, nil) {
			if err != nil {
				return fmt.Errorf("mcp: list tools from %q: %w", conn.Name, err)
			}
			handle := &ToolHandle{
				Server:   conn.Name,
				ToolName: tool.Name,
				Tool:     tool,
			}
			newTools[handle.FullName()] = handle
		}
	}

	m.mu.Lock()
	m.tools = newTools
	m.mu.Unlock()

	return nil
}

// ToolNames returns every tool's full name ("<server>_<tool>").
func (m *Manager) ToolNames() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

// ToolHandle returns the handle for a full tool name.
func (m *Manager) ToolHandle(fullName string) (*ToolHandle, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	handle, ok := m.tools[fullName]
	if !ok {
		return nil, fmt.Errorf("mcp: tool %q not found", fullName)
	}
	return handle, nil
}

// CallTool invokes a tool by its full name ("<server>_<tool>") and returns
// the concatenated text content of the result.
func (m *Manager) CallTool(ctx context.Context, fullName string, arguments map[string]any) (string, error) {
	handle, err := m.ToolHandle(fullName)
	if err != nil {
		return "", err
	}

	m.mu.RLock()
	conn, ok := m.conns[handle.Server]
	m.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("mcp: server %q not connected", handle.Server)
	}

	res, err := conn.Session.CallTool(ctx, &mcp.CallToolParams{
		Name:      handle.ToolName,
		Arguments: arguments,
	})
	if err != nil {
		return "", fmt.Errorf("mcp: call %s: %w", fullName, err)
	}
	if res.IsError {
		return "", fmt.Errorf("mcp: tool %s returned an error: %s", fullName, contentText(res.Content))
	}

	return contentText(res.Content), nil
}

// contentText concatenates all text content blocks in an MCP result. Most
// MCP servers return a single TextContent block containing JSON or plain
// text; non-text blocks (images, embedded resources) are ignored here since
// the agent loop only consumes text.
func contentText(content []mcp.Content) string {
	var out string
	for _, c := range content {
		if tc, ok := c.(*mcp.TextContent); ok {
			if out != "" {
				out += "\n"
			}
			out += tc.Text
		}
	}
	return out
}

// Close closes every connected session.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, conn := range m.conns {
		if err := conn.Session.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("mcp: close %q: %w", name, err)
		}
	}
	m.conns = make(map[string]*ServerConn)
	m.tools = make(map[string]*ToolHandle)
	return firstErr
}

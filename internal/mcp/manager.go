package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Server представляет MCP-сервер (локальный или удалённый)
type Server interface {
	Name() string
	Initialize(ctx context.Context) error
	ListTools(ctx context.Context) ([]ToolDefinition, error)
	CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error)
}

// ToolDefinition описывает инструмент
type ToolDefinition struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
}

// Manager управляет подключениями к MCP-серверам
type Manager struct {
	mu      sync.RWMutex
	servers map[string]Server
	tools   map[string]*ToolHandle // "server.tool" -> handle
}

// ToolHandle связывает инструмент с сервером
type ToolHandle struct {
	Server   string
	ToolName string
	Metadata ToolDefinition
}

// NewManager создаёт новый MCP Manager
func NewManager() *Manager {
	return &Manager{
		servers: make(map[string]Server),
		tools:   make(map[string]*ToolHandle),
	}
}

// RegisterServer добавляет новый сервер
func (m *Manager) RegisterServer(server Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers[server.Name()] = server
	return nil
}

// InitializeAll инициализирует все зарегистрированные серверы
func (m *Manager) InitializeAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, server := range m.servers {
		if err := server.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize server %q: %w", name, err)
		}
	}
	return nil
}

// LoadToolsFromServers загружает все инструменты всех серверов
func (m *Manager) LoadToolsFromServers(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serverName, server := range m.servers {
		tools, err := server.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools from server %q: %w", serverName, err)
		}

		for _, tool := range tools {
			fullName := fmt.Sprintf("%s_%s", serverName, tool.Name)
			m.tools[fullName] = &ToolHandle{
				Server:   serverName,
				ToolName: tool.Name,
				Metadata: tool,
			}
		}
	}
	return nil
}

// GetToolNames возвращает список всех инструментов в формате "server_tool"
func (m *Manager) GetToolNames(ctx context.Context) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

// GetToolMetadata возвращает метаданные инструмента
func (m *Manager) GetToolMetadata(toolFullName string) (*ToolHandle, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	handle, ok := m.tools[toolFullName]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", toolFullName)
	}
	return handle, nil
}

// CallTool вызывает инструмент по полному имени "server_tool"
func (m *Manager) CallTool(ctx context.Context, toolFullName string, arguments map[string]interface{}) (interface{}, error) {
	handle, err := m.GetToolMetadata(toolFullName)
	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	server, ok := m.servers[handle.Server]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("server %q not found", handle.Server)
	}

	return server.CallTool(ctx, handle.ToolName, arguments)
}

// Close закрывает все подключения к серверам
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers = make(map[string]Server)
	m.tools = make(map[string]*ToolHandle)
	return nil
}

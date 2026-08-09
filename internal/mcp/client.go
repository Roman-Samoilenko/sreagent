package mcp

import (
	"context"
	"fmt"
	"sync"
)

type Client struct {
	mu      sync.RWMutex
	servers map[string]MCPServer
}

func NewClient() *Client {
	return &Client{
		servers: make(map[string]MCPServer),
	}
}

// RegisterServer добавляет или заменяет сервер.
func (c *Client) RegisterServer(srv MCPServer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.servers[srv.Name()] = srv
}

// CallTool выполняет вызов инструмента на заданном сервере.
func (c *Client) CallTool(ctx context.Context, call ToolCall) (ToolResult, error) {
	c.mu.RLock()
	srv, ok := c.servers[call.ServerName]
	c.mu.RUnlock()
	if !ok {
		return ToolResult{}, fmt.Errorf("mcp: server %q not found", call.ServerName)
	}

	data, err := srv.HandleToolCall(ctx, call.ToolName, call.Arguments)
	return ToolResult{Content: data, Error: err}, err
}

// ListTools возвращает все инструменты всех серверов в формате "server.tool".
func (c *Client) ListTools(ctx context.Context) []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var names []string
	for _, srv := range c.servers {
		for _, t := range srv.ListTools() {
			names = append(names, fmt.Sprintf("%s.%s", srv.Name(), t))
		}
	}
	return names
}

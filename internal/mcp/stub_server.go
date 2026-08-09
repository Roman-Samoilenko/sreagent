package mcp

import "context"

type StubServer struct {
	name  string
	tools []string
}

func NewStubServer(name string, tools []string) *StubServer {
	return &StubServer{name: name, tools: tools}
}

func (s *StubServer) Name() string                       { return s.name }
func (s *StubServer) Initialize(_ context.Context) error { return nil }
func (s *StubServer) ListTools() []string                { return s.tools }

func (s *StubServer) HandleToolCall(_ context.Context, toolName string, args map[string]any) ([]byte, error) {
	// Возвращаем фиктивный ответ
	return []byte(`{"status":"ok","stub":true}`), nil
}

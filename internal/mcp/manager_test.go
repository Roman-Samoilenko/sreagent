package mcp

import (
	"context"
	"testing"

	gosdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestServer builds a minimal in-process MCP server exposing one "echo"
// tool, for exercising Manager against the real MCP protocol without any
// network or subprocess.
func newTestServer(t *testing.T) *gosdkmcp.Server {
	t.Helper()
	srv := gosdkmcp.NewServer(&gosdkmcp.Implementation{Name: "test-server", Version: "0.0.1"}, nil)

	type echoArgs struct {
		Message string `json:"message"`
	}
	gosdkmcp.AddTool(srv, &gosdkmcp.Tool{
		Name:        "echo",
		Description: "Echoes the message argument back.",
	}, func(_ context.Context, _ *gosdkmcp.CallToolRequest, args echoArgs) (*gosdkmcp.CallToolResult, any, error) {
		return &gosdkmcp.CallToolResult{
			Content: []gosdkmcp.Content{&gosdkmcp.TextContent{Text: "echo: " + args.Message}},
		}, nil, nil
	})

	return srv
}

// connectTestManager wires a Manager to the in-process test server under the
// given logical name, using in-memory transports (no docker, no network).
func connectTestManager(t *testing.T, name string) *Manager {
	t.Helper()
	ctx := context.Background()

	srv := newTestServer(t)
	serverTransport, clientTransport := gosdkmcp.NewInMemoryTransports()

	if _, err := srv.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}

	m := NewManager()
	if err := m.Connect(ctx, name, clientTransport); err != nil {
		t.Fatalf("manager connect: %v", err)
	}
	return m
}

func TestManager_LoadTools_And_CallTool(t *testing.T) {
	ctx := context.Background()
	m := connectTestManager(t, "test")

	if err := m.LoadTools(ctx); err != nil {
		t.Fatalf("LoadTools: %v", err)
	}

	names := m.ToolNames()
	if len(names) != 1 || names[0] != "test_echo" {
		t.Fatalf("ToolNames() = %v, want [test_echo]", names)
	}

	handle, err := m.ToolHandle("test_echo")
	if err != nil {
		t.Fatalf("ToolHandle: %v", err)
	}
	if handle.Server != "test" || handle.ToolName != "echo" {
		t.Fatalf("unexpected handle: %+v", handle)
	}

	result, err := m.CallTool(ctx, "test_echo", map[string]any{"message": "hi"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result != "echo: hi" {
		t.Fatalf("CallTool result = %q, want %q", result, "echo: hi")
	}
}

func TestManager_CallTool_UnknownTool(t *testing.T) {
	ctx := context.Background()
	m := connectTestManager(t, "test")

	if err := m.LoadTools(ctx); err != nil {
		t.Fatalf("LoadTools: %v", err)
	}

	if _, err := m.CallTool(ctx, "test_does_not_exist", nil); err == nil {
		t.Fatal("expected error calling unknown tool, got nil")
	}
}

func TestManager_ToolHandle_NotFound(t *testing.T) {
	m := NewManager()
	if _, err := m.ToolHandle("missing"); err == nil {
		t.Fatal("expected error for missing tool handle, got nil")
	}
}

func TestToolHandle_FullName(t *testing.T) {
	h := &ToolHandle{Server: "github", ToolName: "get_repo"}
	if got, want := h.FullName(), "github_get_repo"; got != want {
		t.Fatalf("FullName() = %q, want %q", got, want)
	}
}

func TestManager_Close(t *testing.T) {
	ctx := context.Background()
	m := connectTestManager(t, "test")
	if err := m.LoadTools(ctx); err != nil {
		t.Fatalf("LoadTools: %v", err)
	}
	if err := m.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if len(m.ToolNames()) != 0 {
		t.Fatal("expected no tools after Close")
	}
}

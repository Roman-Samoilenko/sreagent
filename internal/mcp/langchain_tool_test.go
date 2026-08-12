package mcp

import (
	"context"
	"strings"
	"testing"
)

func TestStripMarkdownFence(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"```json\n{\"a\":1}\n```", `{"a":1}`},
		{"```\n{\"a\":1}\n```", `{"a":1}`},
		{`{"a":1}`, `{"a":1}`},
		{"", ""},
	}
	for _, tc := range cases {
		if got := stripMarkdownFence(tc.in); got != tc.want {
			t.Errorf("stripMarkdownFence(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLangchainTool_Call_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	m := connectTestManager(t, "test")
	if err := m.LoadTools(ctx); err != nil {
		t.Fatalf("LoadTools: %v", err)
	}

	handle, err := m.ToolHandle("test_echo")
	if err != nil {
		t.Fatalf("ToolHandle: %v", err)
	}
	tool := newLangchainTool(m, handle)

	out, err := tool.Call(ctx, "not json")
	if err != nil {
		t.Fatalf("Call returned unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "ERROR:") {
		t.Fatalf("expected ERROR-prefixed output for invalid JSON, got %q", out)
	}
}

func TestLangchainTool_Call_ValidJSON(t *testing.T) {
	ctx := context.Background()
	m := connectTestManager(t, "test")
	if err := m.LoadTools(ctx); err != nil {
		t.Fatalf("LoadTools: %v", err)
	}

	handle, err := m.ToolHandle("test_echo")
	if err != nil {
		t.Fatalf("ToolHandle: %v", err)
	}
	tool := newLangchainTool(m, handle)

	out, err := tool.Call(ctx, `{"message":"hello"}`)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if out != "echo: hello" {
		t.Fatalf("Call output = %q, want %q", out, "echo: hello")
	}
}

func TestLangchainTool_Call_EmptyInput(t *testing.T) {
	// Some tools (e.g. list_collections) take no arguments; an empty input
	// string must not be treated as invalid JSON.
	ctx := context.Background()
	m := connectTestManager(t, "test")
	if err := m.LoadTools(ctx); err != nil {
		t.Fatalf("LoadTools: %v", err)
	}
	handle, err := m.ToolHandle("test_echo")
	if err != nil {
		t.Fatalf("ToolHandle: %v", err)
	}
	tool := newLangchainTool(m, handle)

	if _, err := tool.Call(ctx, ""); err != nil {
		t.Fatalf("Call with empty input returned error: %v", err)
	}
}

func TestBuildTools(t *testing.T) {
	ctx := context.Background()
	m := connectTestManager(t, "test")
	if err := m.LoadTools(ctx); err != nil {
		t.Fatalf("LoadTools: %v", err)
	}

	built := BuildTools(m)
	if len(built) != 1 {
		t.Fatalf("BuildTools() returned %d tools, want 1", len(built))
	}
	if built[0].Name() != "test_echo" {
		t.Fatalf("tool name = %q, want %q", built[0].Name(), "test_echo")
	}
}

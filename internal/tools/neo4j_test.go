package tools

import (
	"context"
	"strings"
	"testing"
)

// Neo4jTool.Call dials out to a real driver only after JSON parsing and
// action-specific validation succeed, so these tests exercise only the
// validation paths that fail before any network I/O happens. Full CRUD
// behavior needs an integration test against a real Neo4j instance.

func newUnconnectedTool(t *testing.T) *Neo4jTool {
	t.Helper()
	// bolt driver construction does not itself dial the server, so this is
	// safe to use for testing input validation without a live database.
	tool, err := NewNeo4jTool("bolt://127.0.0.1:1", "neo4j", "password")
	if err != nil {
		t.Fatalf("NewNeo4jTool: %v", err)
	}
	return tool
}

func TestNeo4jTool_Call_InvalidJSON(t *testing.T) {
	tool := newUnconnectedTool(t)
	_, err := tool.Call(context.Background(), "not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON input")
	}
	if !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNeo4jTool_Call_UnknownAction(t *testing.T) {
	tool := newUnconnectedTool(t)
	_, err := tool.Call(context.Background(), `{"action":"delete_everything"}`)
	if err == nil {
		t.Fatal("expected error for unknown action")
	}
	if !strings.Contains(err.Error(), "unknown action") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNeo4jTool_Call_CreateBug_MissingFields(t *testing.T) {
	tool := newUnconnectedTool(t)
	_, err := tool.Call(context.Background(), `{"action":"create_bug","repo":"myapp"}`)
	if err == nil {
		t.Fatal("expected error for create_bug missing file/description")
	}
	if !strings.Contains(err.Error(), "create_bug requires") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNeo4jTool_Call_CreateFix_MissingFields(t *testing.T) {
	tool := newUnconnectedTool(t)
	_, err := tool.Call(context.Background(), `{"action":"create_fix","repo":"myapp","file":"main.go"}`)
	if err == nil {
		t.Fatal("expected error for create_fix missing suggested_fix")
	}
	if !strings.Contains(err.Error(), "create_fix requires") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNeo4jTool_Call_CreateReport_MissingFields(t *testing.T) {
	tool := newUnconnectedTool(t)
	_, err := tool.Call(context.Background(), `{"action":"create_report","repo":"myapp","file":"main.go"}`)
	if err == nil {
		t.Fatal("expected error for create_report missing fields")
	}
	if !strings.Contains(err.Error(), "create_report requires") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNeo4jTool_Name(t *testing.T) {
	tool := newUnconnectedTool(t)
	if tool.Name() != "knowledge_graph" {
		t.Fatalf("Name() = %q, want %q", tool.Name(), "knowledge_graph")
	}
}

package tools

import (
	"context"
	"testing"
)

func TestReposTool_Call(t *testing.T) {
	repos := []string{"github.com/a/b", "github.com/c/d"}
	tool := NewReposTool(repos)

	out, err := tool.Call(context.Background(), "")
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	want := "github.com/a/b\ngithub.com/c/d"
	if out != want {
		t.Fatalf("Call() = %q, want %q", out, want)
	}
}

func TestReposTool_Call_Empty(t *testing.T) {
	tool := NewReposTool(nil)
	out, err := tool.Call(context.Background(), "")
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	if out != "" {
		t.Fatalf("Call() = %q, want empty string", out)
	}
}

func TestReposTool_Name(t *testing.T) {
	tool := NewReposTool(nil)
	if tool.Name() != "list_configured_repos" {
		t.Fatalf("Name() = %q", tool.Name())
	}
}

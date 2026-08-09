package tools

import (
	"context"
	"strings"
)

type ReposTool struct {
	repos []string
}

func NewReposTool(repos []string) *ReposTool {
	return &ReposTool{repos: repos}
}

func (r *ReposTool) Name() string {
	return "list_configured_repos"
}

func (r *ReposTool) Description() string {
	return `Returns the list of monitored repository URLs from the configuration. 
Use this tool FIRST when you need to analyze a repository, to find available owner/repo pairs. 
No input required. Output is a list of URLs, e.g. "github.com/owner/repo".`
}

func (r *ReposTool) Call(ctx context.Context, input string) (string, error) {
	return strings.Join(r.repos, "\n"), nil
}

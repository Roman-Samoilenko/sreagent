package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/roman-samoilenko/sreagent/internal/mcp"
	"github.com/tmc/langchaingo/tools"
)

// GitHubMCPTool — обёртка для инструментов MCP GitHub.
type GitHubMCPTool struct {
	server      *mcp.GitHubServer
	toolName    string
	description string
}

// NewGitHubMCPTool создаёт инструмент из конкретного метода MCP‑сервера.
func NewGitHubMCPTool(server *mcp.GitHubServer, toolName string) (tools.Tool, error) {
	desc, ok := descriptions[toolName]
	if !ok {
		return nil, fmt.Errorf("unknown tool %s", toolName)
	}
	return &GitHubMCPTool{
		server:      server,
		toolName:    toolName,
		description: desc,
	}, nil
}

func (t *GitHubMCPTool) Name() string {
	return "github_" + t.toolName
}

func (t *GitHubMCPTool) Description() string {
	return t.description
}

func (t *GitHubMCPTool) Call(ctx context.Context, input string) (string, error) {
	// Парсим входной JSON в map, как ожидает MCP‑сервер
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		// Если не JSON, пробуем передать как {"query": input} для search_code
		args = map[string]interface{}{"query": input}
	}
	res, err := t.server.CallTool(ctx, t.toolName, args)
	if err != nil {
		return "", err
	}
	// Сериализуем результат обратно в строку
	out, _ := json.MarshalIndent(res, "", "  ")
	return string(out), nil
}

// Описания, которые понимает LLM и чётко указывают на ReAct-формат.
var descriptions = map[string]string{
	"get_repo": `Get repository info. 
Input MUST be JSON: {"owner": "owner-name", "repo": "repo-name"}
Output: JSON with name, full_name, description, stars, language, etc.`,

	"list_files": `List files in a repository directory.
Input: {"owner": "...", "repo": "...", "path": "optional/dir", "recursive": true/false}
Output: array of {name, path, type}.`,

	"get_file_content": `Read file content from GitHub.
Input: {"owner": "...", "repo": "...", "path": "file/path"}
Output: {path, content, size}. Content is the full text.`,

	"search_code": `Search code in a repository.
Input: {"owner": "...", "repo": "...", "query": "search terms"}
Output: {total_count, matches: [{name, path}]}`,

	"list_issues": `List repository issues.
Input: {"owner": "...", "repo": "...", "state": "open|closed|all"}
Output: array of {number, title, body, state}.`,
}

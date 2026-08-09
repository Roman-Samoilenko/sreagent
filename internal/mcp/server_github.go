package mcp

import (
	"context"
	"fmt"

	"github.com/google/go-github/v63/github"
	"golang.org/x/oauth2"
)

// GitHubServer реализует MCP Server для GitHub
type GitHubServer struct {
	name   string
	client *github.Client
}

// NewGitHubServer создаёт новый GitHub MCP Server
func NewGitHubServer(token string) (*GitHubServer, error) {
	if token == "" {
		return nil, fmt.Errorf("github token is required")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &GitHubServer{
		name:   "github",
		client: github.NewClient(tc),
	}, nil
}

// Name возвращает имя сервера
func (s *GitHubServer) Name() string {
	return s.name
}

// Initialize инициализирует подключение
func (s *GitHubServer) Initialize(ctx context.Context) error {
	// Проверяем подключение к API
	_, _, err := s.client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to authenticate with GitHub: %w", err)
	}
	return nil
}

// ListTools возвращает список доступных инструментов
func (s *GitHubServer) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	return []ToolDefinition{
		{
			Name:        "get_repo",
			Description: `Get repository info. Use list_configured_repos first to get available repos. Input MUST be JSON: {"owner": "owner-name", "repo": "repo-name"}. Output: JSON with name, full_name, description, stars, language, etc.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string", "description": "Owner of the repository"},
					"repo":  map[string]interface{}{"type": "string", "description": "Repository name"},
				},
				"required": []string{"owner", "repo"},
			},
		},
		{
			Name:        "list_files",
			Description: `List files in a repo directory. Get repository info. Use list_configured_repos first to get available repos. Input JSON: {"owner":"...", "repo":"...", "path":"optional/dir", "recursive":true/false}. Output: array of {name, path, type}.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner":     map[string]interface{}{"type": "string"},
					"repo":      map[string]interface{}{"type": "string"},
					"path":      map[string]interface{}{"type": "string", "description": "Optional: directory path"},
					"recursive": map[string]interface{}{"type": "boolean", "description": "Recursive listing"},
				},
				"required": []string{"owner", "repo"},
			},
		},
		{
			Name:        "get_file_content",
			Description: `Read file content from GitHub. Get repository info. Use list_configured_repos first to get available repos. Input JSON: {"owner":"...", "repo":"...", "path":"file/path"}. Output: {path, content, size}.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string"},
					"repo":  map[string]interface{}{"type": "string"},
					"path":  map[string]interface{}{"type": "string"},
				},
				"required": []string{"owner", "repo", "path"},
			},
		},
		{
			Name:        "search_code",
			Description: `Search code in a repository.Get repository info. Use list_configured_repos first to get available repos. Input JSON: {"owner":"...", "repo":"...", "query":"search terms"}. Output: {total_count, matches: [{name, path}]}.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string"},
					"repo":  map[string]interface{}{"type": "string"},
					"query": map[string]interface{}{"type": "string"},
				},
				"required": []string{"owner", "repo", "query"},
			},
		},
		{
			Name:        "list_issues",
			Description: `List repository issues. Get repository info. Use list_configured_repos first to get available repos. Input JSON: {"owner":"...", "repo":"...", "state":"open|closed|all"}. Output: array of {number, title, body, state}.`,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string"},
					"repo":  map[string]interface{}{"type": "string"},
					"state": map[string]interface{}{"type": "string", "enum": []string{"open", "closed", "all"}},
				},
				"required": []string{"owner", "repo"},
			},
		},
	}, nil
}

// CallTool вызывает инструмент
func (s *GitHubServer) CallTool(ctx context.Context, toolName string, arguments map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "get_repo":
		return s.getRepo(ctx, arguments)
	case "list_files":
		return s.listFiles(ctx, arguments)
	case "get_file_content":
		return s.getFileContent(ctx, arguments)
	case "search_code":
		return s.searchCode(ctx, arguments)
	case "list_issues":
		return s.listIssues(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// getRepo получает информацию о репозитории
func (s *GitHubServer) getRepo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, ok := args["owner"].(string)
	if !ok {
		return nil, fmt.Errorf("owner is required and must be a string")
	}
	repo, ok := args["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required and must be a string")
	}

	r, _, err := s.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo: %w", err)
	}

	return map[string]interface{}{
		"name":           r.Name,
		"full_name":      r.FullName,
		"description":    r.Description,
		"url":            r.HTMLURL,
		"stars":          r.StargazersCount,
		"forks":          r.ForksCount,
		"language":       r.Language,
		"default_branch": r.DefaultBranch,
	}, nil
}

// listFiles получает список файлов
func (s *GitHubServer) listFiles(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, ok := args["owner"].(string)
	if !ok {
		return nil, fmt.Errorf("owner is required")
	}
	repo, ok := args["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}
	path, _ := args["path"].(string)

	opts := &github.RepositoryContentGetOptions{}
	_, directoryContent, _, err := s.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	var files []map[string]interface{}
	for _, file := range directoryContent {
		files = append(files, map[string]interface{}{
			"name": file.Name,
			"path": file.Path,
			"type": file.Type,
			"url":  file.HTMLURL,
		})
	}

	return files, nil
}

// getFileContent получает содержимое файла
func (s *GitHubServer) getFileContent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, ok := args["owner"].(string)
	if !ok {
		return nil, fmt.Errorf("owner is required")
	}
	repo, ok := args["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path is required")
	}

	fileContent, _, _, err := s.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return map[string]interface{}{
		"path":    fileContent.Path,
		"content": content,
		"size":    fileContent.Size,
	}, nil
}

// searchCode ищет код в репозитории
func (s *GitHubServer) searchCode(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, ok := args["owner"].(string)
	if !ok {
		return nil, fmt.Errorf("owner is required")
	}
	repo, ok := args["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required")
	}

	searchQuery := fmt.Sprintf("%s repo:%s/%s", query, owner, repo)
	result, _, err := s.client.Search.Code(ctx, searchQuery, nil)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var matches []map[string]interface{}
	for _, item := range result.CodeResults {
		matches = append(matches, map[string]interface{}{
			"name": item.Name,
			"path": item.Path,
			"url":  item.HTMLURL,
		})
	}

	return map[string]interface{}{
		"total_count": result.Total,
		"matches":     matches,
	}, nil
}

// listIssues получает список issues
func (s *GitHubServer) listIssues(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, ok := args["owner"].(string)
	if !ok {
		return nil, fmt.Errorf("owner is required")
	}
	repo, ok := args["repo"].(string)
	if !ok {
		return nil, fmt.Errorf("repo is required")
	}
	state, _ := args["state"].(string)
	if state == "" {
		state = "open"
	}

	opts := &github.IssueListByRepoOptions{
		State: state,
		ListOptions: github.ListOptions{
			PerPage: 20,
		},
	}

	issues, _, err := s.client.Issues.ListByRepo(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list issues: %w", err)
	}

	var result []map[string]interface{}
	for _, issue := range issues {
		result = append(result, map[string]interface{}{
			"number": issue.Number,
			"title":  issue.Title,
			"body":   issue.Body,
			"state":  issue.State,
			"url":    issue.HTMLURL,
		})
	}

	return result, nil
}

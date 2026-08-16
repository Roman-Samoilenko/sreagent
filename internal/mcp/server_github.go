package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/go-github/v63/github"
	"golang.org/x/oauth2"
)

type GitHubServer struct {
	name   string
	client *github.Client
}

func NewGitHubServer(token string) (*GitHubServer, error) {
	if token == "" {
		return nil, errors.New("github token is required")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return &GitHubServer{
		name:   "github",
		client: github.NewClient(tc),
	}, nil
}

func (s *GitHubServer) Name() string { return s.name }

func (s *GitHubServer) Initialize(ctx context.Context) error {
	_, _, err := s.client.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to authenticate with GitHub: %w", err)
	}
	return nil
}

func (s *GitHubServer) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	return []ToolDefinition{
		{
			Name:        "get_repo",
			Description: "Get repository info. Use list_configured_repos first to get available repos. Input JSON: {\"owner\":\"...\", \"repo\":\"...\"}. Output: JSON with name, full_name, description, stars, language, etc.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string"},
					"repo":  map[string]interface{}{"type": "string"},
				},
				"required": []string{"owner", "repo"},
			},
		},
		{
			Name:        "list_files",
			Description: "List files in a repository directory (not a file). Use list_configured_repos first. Input JSON: {\"owner\":\"...\", \"repo\":\"...\", \"path\":\"optional/dir\"}. Output: array of {name, path, type}.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"owner": map[string]interface{}{"type": "string"},
					"repo":  map[string]interface{}{"type": "string"},
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Optional: directory path (default root)",
					},
				},
				"required": []string{"owner", "repo"},
			},
		},
		{
			Name:        "get_file_content",
			Description: "Read file content from GitHub. Use list_configured_repos first. Input JSON: {\"owner\":\"...\", \"repo\":\"...\", \"path\":\"file/path\"}. Output: {path, content, size}.",
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
			Description: "Search code in a repository. Use list_configured_repos first. Input JSON: {\"owner\":\"...\", \"repo\":\"...\", \"query\":\"search terms\"}. Output: {total_count, matches: [{name, path}]}.",
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
			Description: "List repository issues. Use list_configured_repos first. Input JSON: {\"owner\":\"...\", \"repo\":\"...\", \"state\":\"open|closed|all\"}. Output: array of {number, title, body, state}.",
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

func (s *GitHubServer) CallTool(
	ctx context.Context,
	toolName string,
	arguments map[string]interface{},
) (interface{}, error) {
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

func (s *GitHubServer) getRepo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, repo, err := parseOwnerRepo(args)
	if err != nil {
		return nil, err
	}
	r, _, err := s.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo: %w", err)
	}
	return map[string]interface{}{
		"name":           r.GetName(),
		"full_name":      r.GetFullName(),
		"description":    r.GetDescription(),
		"url":            r.GetHTMLURL(),
		"stars":          r.GetStargazersCount(),
		"forks":          r.GetForksCount(),
		"language":       r.GetLanguage(),
		"default_branch": r.GetDefaultBranch(),
	}, nil
}
func (s *GitHubServer) listFiles(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, repo, err := parseOwnerRepo(args)
	if err != nil {
		return nil, err
	}
	path, _ := args["path"].(string)

	opts := &github.RepositoryContentGetOptions{}
	fileContent, directoryContent, _, err := s.client.Repositories.GetContents(ctx, owner, repo, path, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents: %w", err)
	}

	if fileContent != nil {
		return nil, fmt.Errorf("'%s' is a file, not a directory. Use get_file_content to read it", path)
	}

	var files []map[string]interface{}
	for _, item := range directoryContent {
		files = append(files, map[string]interface{}{
			"name": item.GetName(),
			"path": item.GetPath(),
			"type": item.GetType(),
			"url":  item.GetHTMLURL(),
		})
	}
	return files, nil
}
func (s *GitHubServer) getFileContent(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, repo, err := parseOwnerRepo(args)
	if err != nil {
		return nil, err
	}
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return nil, errors.New("path is required")
	}

	fileContent, _, _, err := s.client.Repositories.GetContents(ctx, owner, repo, path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}
	if fileContent == nil {
		return nil, errors.New("file not found")
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return nil, fmt.Errorf("failed to decode file content: %w", err)
	}

	return map[string]interface{}{
		"path":    fileContent.GetPath(),
		"content": content,
		"size":    fileContent.GetSize(),
	}, nil
}

func (s *GitHubServer) searchCode(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, repo, err := parseOwnerRepo(args)
	if err != nil {
		return nil, err
	}
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, errors.New("query is required")
	}

	searchQuery := fmt.Sprintf("%s repo:%s/%s", query, owner, repo)
	result, _, err := s.client.Search.Code(ctx, searchQuery, &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 30},
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	var matches []map[string]interface{}
	for _, item := range result.CodeResults {
		matches = append(matches, map[string]interface{}{
			"name": item.GetName(),
			"path": item.GetPath(),
			"url":  item.GetHTMLURL(),
		})
	}
	return map[string]interface{}{
		"total_count": result.GetTotal(),
		"matches":     matches,
	}, nil
}

func (s *GitHubServer) listIssues(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	owner, repo, err := parseOwnerRepo(args)
	if err != nil {
		return nil, err
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
			"number": issue.GetNumber(),
			"title":  issue.GetTitle(),
			"body":   issue.GetBody(),
			"state":  issue.GetState(),
			"url":    issue.GetHTMLURL(),
		})
	}
	return result, nil
}

// parseOwnerRepo извлекает owner и repo из аргументов.
func parseOwnerRepo(args map[string]interface{}) (string, string, error) {
	owner, ok := args["owner"].(string)
	if !ok || owner == "" {
		return "", "", errors.New("owner is required")
	}
	repo, ok := args["repo"].(string)
	if !ok || repo == "" {
		return "", "", errors.New("repo is required")
	}
	return owner, repo, nil
}

package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jTool struct {
	driver neo4j.DriverWithContext
}

func NewNeo4jTool(uri, user, password string) (*Neo4jTool, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, password, ""))
	if err != nil {
		return nil, err
	}
	return &Neo4jTool{driver: driver}, nil
}

func (n *Neo4jTool) Name() string { return "knowledge_graph" }
func (n *Neo4jTool) Description() string {
	return `Stores incident analysis results into a Neo4j knowledge graph.
Input MUST be a JSON object with an "action" field and the required parameters:

1. "create_bug" - records a bug/vulnerability in a file.
   Required: "repo", "file", "description"
   Returns: {"status":"bug_created"}
   Example: {"action":"create_bug","repo":"myapp","file":"main.go","description":"SQL injection in login"}

2. "create_fix" - links a suggested fix to the latest unresolved bug in the same file.
   Required: "repo", "file", "suggested_fix"
   Returns: {"status":"fix_linked"}
   Example: {"action":"create_fix","repo":"myapp","file":"main.go","suggested_fix":"Use parameterized queries"}

3. "create_report" - attaches a detailed report to a specific bug.
   Required: "repo", "file", "description" (must match exactly the bug's description), "report_id", "content"
   Returns: {"status":"report_created"}
   Example: {"action":"create_report","repo":"myapp","file":"main.go","description":"SQL injection in login","report_id":"RPT-001","content":"Full analysis..."}

Graph relationships:
- Every bug is linked to a File and a Repo.
- A fix is connected to the most recent unresolved bug in that file.
- A report is linked to the matching bug.

Typical workflow: "create_bug" -> "create_fix" or "create_report".`
}

func (n *Neo4jTool) Call(ctx context.Context, input string) (string, error) {
	var req struct {
		Action      string `json:"action"`
		Repo        string `json:"repo,omitempty"`
		File        string `json:"file,omitempty"`
		Description string `json:"description,omitempty"`
		Fix         string `json:"suggested_fix,omitempty"`
		ReportID    string `json:"report_id,omitempty"`
		Content     string `json:"content,omitempty"`
	}
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("Neo4jTool: invalid input: %w", err)
	}

	session := n.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	switch req.Action {
	case "create_bug":
		if req.Repo == "" || req.File == "" || req.Description == "" {
			return "", errors.New("Neo4jTool: create_bug requires repo, file, description")
		}
		_, err := session.Run(ctx, `
            MERGE (r:Repo {name: $repo})
            MERGE (f:File {path: $file, repo: $repo})
            MERGE (r)-[:CONTAINS]->(f)
            CREATE (b:Bug {description: $description, file: $file, repo: $repo, timestamp: datetime()})
            CREATE (f)-[:HAS_BUG]->(b)
            RETURN id(b)
        `, map[string]any{
			"repo":        req.Repo,
			"file":        req.File,
			"description": req.Description,
		})
		if err != nil {
			return "", fmt.Errorf("create bug: %w", err)
		}
		return `{"status":"bug_created"}`, nil

	case "create_fix":
		if req.Repo == "" || req.File == "" || req.Fix == "" {
			return "", errors.New("Neo4jTool: create_fix requires repo, file, suggested_fix")
		}
		_, err := session.Run(ctx, `
            MATCH (b:Bug {file: $file, repo: $repo})
            WHERE NOT EXISTS((b)-[:FIXED_BY]->())
            WITH b ORDER BY b.timestamp DESC LIMIT 1
            CREATE (f:Fix {suggested_fix: $fix, timestamp: datetime()})
            CREATE (b)-[:FIXED_BY]->(f)
            RETURN f
        `, map[string]any{
			"repo": req.Repo,
			"file": req.File,
			"fix":  req.Fix,
		})
		if err != nil {
			return "", fmt.Errorf("create fix: %w", err)
		}
		return `{"status":"fix_linked"}`, nil

	case "create_report":
		if req.Repo == "" || req.File == "" || req.Description == "" || req.ReportID == "" || req.Content == "" {
			return "", errors.New("Neo4jTool: create_report requires repo, file, description, report_id, content")
		}
		_, err := session.Run(ctx, `
            MATCH (b:Bug {file: $file, repo: $repo, description: $description})
            CREATE (r:Report {report_id: $report_id, content: $content, timestamp: datetime()})
            CREATE (b)-[:HAS_REPORT]->(r)
            RETURN r
        `, map[string]any{
			"repo":        req.Repo,
			"file":        req.File,
			"description": req.Description,
			"report_id":   req.ReportID,
			"content":     req.Content,
		})
		if err != nil {
			return "", fmt.Errorf("create report: %w", err)
		}
		return `{"status":"report_created"}`, nil

	default:
		return "", fmt.Errorf("Neo4jTool: unknown action %q", req.Action)
	}
}

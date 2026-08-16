package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jServer реализует MCP Server для Neo4j.
type Neo4jServer struct {
	name   string
	driver neo4j.DriverWithContext
}

// NewNeo4jServer создаёт новый Neo4j MCP Server.
func NewNeo4jServer(uri, username, password string) (*Neo4jServer, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}
	return &Neo4jServer{
		name:   "neo4j",
		driver: driver,
	}, nil
}

// Name возвращает имя сервера.
func (s *Neo4jServer) Name() string {
	return s.name
}

// Initialize инициализирует подключение.
func (s *Neo4jServer) Initialize(ctx context.Context) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	_, err := session.Run(ctx, "RETURN 1", nil)
	return err
}

// ListTools возвращает список доступных инструментов.
func (s *Neo4jServer) ListTools(ctx context.Context) ([]ToolDefinition, error) {
	return []ToolDefinition{
		{
			Name:        "create_bug",
			Description: "Создать узел Bug. Аргументы: {\"repo\": \"...\", \"file\": \"...\", \"description\": \"...\"}",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo":        map[string]interface{}{"type": "string"},
					"file":        map[string]interface{}{"type": "string"},
					"description": map[string]interface{}{"type": "string"},
				},
				"required": []string{"repo", "file", "description"},
			},
		},
		{
			Name:        "create_fix",
			Description: "Создать узел Fix и связать с Bug. Аргументы: {\"repo\": \"...\", \"file\": \"...\", \"fix\": \"...\"}",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo": map[string]interface{}{"type": "string"},
					"file": map[string]interface{}{"type": "string"},
					"fix":  map[string]interface{}{"type": "string"},
				},
				"required": []string{"repo", "file", "fix"},
			},
		},
		{
			Name:        "create_report",
			Description: "Создать узел Report. Аргументы: {\"repo\": \"...\", \"file\": \"...\", \"description\": \"...\", \"report_id\": \"...\", \"content\": \"...\"}",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"repo":        map[string]interface{}{"type": "string"},
					"file":        map[string]interface{}{"type": "string"},
					"description": map[string]interface{}{"type": "string"},
					"report_id":   map[string]interface{}{"type": "string"},
					"content":     map[string]interface{}{"type": "string"},
				},
				"required": []string{"repo", "file", "description", "report_id", "content"},
			},
		},
		{
			Name:        "query",
			Description: "Выполнить Cypher-запрос к Neo4j. Аргументы: {\"cypher\": \"...\"}",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cypher": map[string]interface{}{"type": "string", "description": "Cypher query"},
				},
				"required": []string{"cypher"},
			},
		},
	}, nil
}

// CallTool вызывает инструмент.
func (s *Neo4jServer) CallTool(
	ctx context.Context,
	toolName string,
	arguments map[string]interface{},
) (interface{}, error) {
	switch toolName {
	case "create_bug":
		return s.createBug(ctx, arguments)
	case "create_fix":
		return s.createFix(ctx, arguments)
	case "create_report":
		return s.createReport(ctx, arguments)
	case "query":
		return s.query(ctx, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// createBug создаёт узел Bug в графе.
func (s *Neo4jServer) createBug(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	repo, _ := args["repo"].(string)
	file, _ := args["file"].(string)
	description, _ := args["description"].(string)

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MERGE (r:Repo {name: $repo})
		MERGE (f:File {path: $file, repo: $repo})
		MERGE (r)-[:CONTAINS]->(f)
		CREATE (b:Bug {description: $description, file: $file, repo: $repo, timestamp: datetime()})
		CREATE (f)-[:HAS_BUG]->(b)
		RETURN id(b) as bug_id
	`, map[string]any{
		"repo":        repo,
		"file":        file,
		"description": description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bug: %w", err)
	}

	record, err := result.Single(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	bugID, _ := record.Get("bug_id")
	return map[string]interface{}{
		"status":  "created",
		"bug_id":  bugID,
		"message": fmt.Sprintf("Bug created with ID %v", bugID),
	}, nil
}

// createFix создаёт узел Fix и связывает с последним Bug.
func (s *Neo4jServer) createFix(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	repo, _ := args["repo"].(string)
	file, _ := args["file"].(string)
	fix, _ := args["fix"].(string)

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH (b:Bug {file: $file, repo: $repo})
		WHERE NOT EXISTS((b)-[:FIXED_BY]->())
		WITH b ORDER BY b.timestamp DESC LIMIT 1
		CREATE (f:Fix {suggested_fix: $fix, timestamp: datetime()})
		CREATE (b)-[:FIXED_BY]->(f)
		RETURN id(f) as fix_id
	`, map[string]any{
		"repo": repo,
		"file": file,
		"fix":  fix,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fix: %w", err)
	}

	record, err := result.Single(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	fixID, _ := record.Get("fix_id")
	return map[string]interface{}{
		"status":  "linked",
		"fix_id":  fixID,
		"message": fmt.Sprintf("Fix created and linked with ID %v", fixID),
	}, nil
}

// createReport создаёт узел Report.
func (s *Neo4jServer) createReport(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	repo, _ := args["repo"].(string)
	file, _ := args["file"].(string)
	description, _ := args["description"].(string)
	reportID, _ := args["report_id"].(string)
	content, _ := args["content"].(string)

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.Run(ctx, `
		MATCH (b:Bug {file: $file, repo: $repo, description: $description})
		CREATE (r:Report {report_id: $report_id, content: $content, timestamp: datetime()})
		CREATE (b)-[:HAS_REPORT]->(r)
		RETURN id(r) as report_id
	`, map[string]any{
		"repo":        repo,
		"file":        file,
		"description": description,
		"report_id":   reportID,
		"content":     content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create report: %w", err)
	}

	record, err := result.Single(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}

	reportNodeID, _ := record.Get("report_id")
	return map[string]interface{}{
		"status":    "created",
		"report_id": reportNodeID,
		"message":   fmt.Sprintf("Report created with ID %v", reportNodeID),
	}, nil
}

// query выполняет произвольный Cypher-запрос.
func (s *Neo4jServer) query(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	cypher, ok := args["cypher"].(string)
	if !ok {
		return nil, errors.New("cypher query is required")
	}

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, cypher, nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	records, err := result.Collect(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect results: %w", err)
	}

	var resultRecords []map[string]interface{}
	for _, record := range records {
		resultRecords = append(resultRecords, record.AsMap())
	}

	return map[string]interface{}{
		"count":   len(resultRecords),
		"records": resultRecords,
	}, nil
}

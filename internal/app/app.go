package app

import (
	"context"
	"fmt"

	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/internal/core"
	"github.com/roman-samoilenko/sreagent/internal/llm"
	"github.com/roman-samoilenko/sreagent/internal/mcp"
	mytools "github.com/roman-samoilenko/sreagent/internal/tools"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

type App struct {
	Config       *config.Config
	Orchestrator core.Orchestrator
	MCPManager   *mcp.Manager

	llmModel llms.Model
	allTools []tools.Tool
}

func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	// 1. LLM
	llmModel, err := llm.NewLLM(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	// 2. MCP Manager: connects to real, external MCP servers only.
	mcpManager := mcp.NewManager()

	if cfg.MCP.GitHub.Enabled {
		if cfg.GitHubToken == "" {
			logger.Warn("GitHub MCP enabled but GITHUB_TOKEN is not set; skipping")
		} else {
			transport, err := mcp.GitHubDockerTransport(cfg.GitHubToken, cfg.MCP.GitHub.Toolsets)
			if err != nil {
				logger.Warn("Failed to build GitHub MCP transport", "error", err)
			} else if err := mcpManager.Connect(ctx, "github", transport); err != nil {
				logger.Warn("Failed to connect to GitHub MCP server", "error", err)
			} else {
				logger.Info("Connected to GitHub MCP server")
			}
		}
	}

	if cfg.MCP.Qdrant.Enabled {
		transport, err := mcp.QdrantHTTPTransport(cfg.MCP.Qdrant.URL)
		if err != nil {
			logger.Warn("Failed to build Qdrant MCP transport", "error", err)
		} else if err := mcpManager.Connect(ctx, "qdrant", transport); err != nil {
			logger.Warn("Failed to connect to Qdrant MCP server", "error", err)
		} else {
			logger.Info("Connected to Qdrant MCP server", "url", cfg.MCP.Qdrant.URL)
		}
	}

	if err := mcpManager.LoadTools(ctx); err != nil {
		logger.Warn("Failed to load tools from MCP servers", "error", err)
	}

	mcpTools := mcp.BuildTools(mcpManager)
	logger.Info("Loaded MCP tools", "count", len(mcpTools))

	// 3. Non-MCP tools: repo list helper + direct Neo4j knowledge-graph tool.
	allTools := []tools.Tool{mytools.NewReposTool(cfg.Repositories)}
	allTools = append(allTools, mcpTools...)

	if cfg.Neo4j.Enabled {
		neo4jTool, err := mytools.NewNeo4jTool(cfg.Neo4j.URI, cfg.Neo4j.User, cfg.Neo4jPassword)
		if err != nil {
			logger.Warn("Failed to create Neo4j tool", "error", err)
		} else {
			allTools = append(allTools, neo4jTool)
			logger.Info("Neo4j knowledge-graph tool registered")
		}
	}

	logger.Info("Registered tools (total)", "count", len(allTools))

	// 4. Orchestrator
	orchestrator, err := core.NewLangchainOrchestrator(llmModel, allTools, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	return &App{
		Config:       cfg,
		Orchestrator: orchestrator,
		MCPManager:   mcpManager,
		llmModel:     llmModel,
		allTools:     allTools,
	}, nil
}

func (a *App) NewOrchestrator() (core.Orchestrator, error) {
	return core.NewLangchainOrchestrator(a.llmModel, a.allTools, a.Config)
}

func (a *App) Close() error {
	if a.MCPManager != nil {
		return a.MCPManager.Close()
	}
	return nil
}

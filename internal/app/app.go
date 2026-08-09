package app

import (
	"context"
	"fmt"

	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/internal/core"
	"github.com/roman-samoilenko/sreagent/internal/llm"
	"github.com/roman-samoilenko/sreagent/internal/mcp"
	"github.com/roman-samoilenko/sreagent/internal/memory"
	mytools "github.com/roman-samoilenko/sreagent/internal/tools"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
	"github.com/tmc/langchaingo/vectorstores"
)

type App struct {
	Config       *config.Config
	Orchestrator core.Orchestrator
	QdrantStore  vectorstores.VectorStore
	MCPManager   *mcp.Manager
}

func New(cfg *config.Config) (*App, error) {
	ctx := context.Background()

	// 1. Инициализируем LLM
	llmModel, err := llm.NewLLM(&cfg.LLM)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	// 2. Инициализируем MCP Manager
	mcpManager := mcp.NewManager()

	// 3. Регистрируем GitHub Server
	if cfg.GitHubToken != "" {
		githubServer, err := mcp.NewGitHubServer(cfg.GitHubToken)
		if err != nil {
			logger.Warn("Failed to create GitHub server", "error", err)
		} else {
			if err := mcpManager.RegisterServer(githubServer); err != nil {
				logger.Warn("Failed to register GitHub server", "error", err)
			} else {
				logger.Info("GitHub server registered")
			}
		}
	}

	// 4. Регистрируем Neo4j Server
	if neo4jURL, err := cfg.GetMCPServerURL("neo4j"); err == nil {
		neo4jServer, err := mcp.NewNeo4jServer(neo4jURL, "neo4j", cfg.Neo4jPassword)
		if err != nil {
			logger.Warn("Failed to create Neo4j server", "error", err)
		} else {
			if err := mcpManager.RegisterServer(neo4jServer); err != nil {
				logger.Warn("Failed to register Neo4j server", "error", err)
			} else {
				logger.Info("Neo4j server registered")
			}
		}
	}

	// 5. Регистрируем Qdrant Server
	var qdrantStore vectorstores.VectorStore

	if qdrantURL, err := cfg.GetMCPServerURL("qdrant"); err == nil {
		qdrantServer, err := mcp.NewQdrantServer(qdrantURL)
		if err != nil {
			logger.Warn("Failed to create Qdrant server", "error", err)
		} else {
			if err := mcpManager.RegisterServer(qdrantServer); err != nil {
				logger.Warn("Failed to register Qdrant server", "error", err)
			} else {
				logger.Info("Qdrant server registered")
				// Пытаемся создать vector store если Qdrant доступен
				embeddingClient, err := openai.New(
					openai.WithToken(cfg.LLM.OpenRouterAPIKey),
					openai.WithModel("text-embedding-ada-002"),
					openai.WithBaseURL("https://api.openai.com/v1"),
				)
				if err == nil {
					qdrantStore, err = memory.NewQdrantStore("incident_docs", embeddingClient, qdrantURL)
					if err != nil {
						logger.Warn("Failed to create Qdrant store", "error", err)
					}
				}
			}
		}
	}

	// 6. Инициализируем все серверы
	if err := mcpManager.InitializeAll(ctx); err != nil {
		logger.Warn("Failed to initialize some MCP servers", "error", err)
	}

	// 7. Создаём инструменты из MCP
	factory := mcp.NewMCPToolFactory(mcpManager)
	mcpTools, err := factory.BuildTools(ctx)
	if err != nil {
		logger.Warn("Failed to build MCP tools", "error", err)
		mcpTools = []tools.Tool{}
	}

	// Добавляем инструмент для получения списка репозиториев
	reposTool := mytools.NewReposTool(cfg.Repositories)
	allTools := append([]tools.Tool{reposTool}, mcpTools...)

	logger.Info("Registered tools (including repos)", "count", len(allTools))

	// 8. Создаём оркестратор с инструментами
	orchestrator, err := core.NewLangchainOrchestrator(llmModel, allTools, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	return &App{
		Config:       cfg,
		Orchestrator: orchestrator,
		QdrantStore:  qdrantStore,
		MCPManager:   mcpManager,
	}, nil
}

func (a *App) Close() error {
	if a.MCPManager != nil {
		return a.MCPManager.Close()
	}
	return nil
}

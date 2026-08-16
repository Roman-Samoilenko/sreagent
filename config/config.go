package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	LLM             LLMConfig      `yaml:"llm"`
	Repositories    []string       `yaml:"repositories"`
	MCP             MCPConfig      `yaml:"mcp"`
	Neo4j           Neo4jConfig    `yaml:"neo4j"`
	Agents          AgentsConfig   `yaml:"agents"`
	Telegram        TelegramConfig `yaml:"telegram"`
	ScanInterval    string         `yaml:"scan_interval"`
	MaxReActSteps   int            `yaml:"max_react_steps"`
	LogLevel        string         `yaml:"log_level"`
	FilesExtensions []string       `yaml:"files_extensions"`
	ExcludeDirs     []string       `yaml:"exclude_dirs"`
	GitHubToken     string         `yaml:"-"`
	Neo4jPassword   string         `yaml:"-"`
}

type LLMConfig struct {
	Provider               string   `yaml:"provider"`                  // "openrouter" или "local"
	OpenRouterModels       []string `yaml:"open_router_models"`        // список доступных моделей
	OpenRouterCurrentModel string   `yaml:"open_router_current_model"` // модель по умолчанию
	OpenRouterAPIKey       string   `yaml:"-"`                         // только из env
}

// MCPConfig configures connections to real, external MCP servers.
type MCPConfig struct {
	// GitHub server, launched as a stdio subprocess:
	//   docker run -i --rm -e GITHUB_PERSONAL_ACCESS_TOKEN ghcr.io/github/github-mcp-server
	GitHub MCPGitHubConfig `yaml:"github"`

	Qdrant MCPQdrantConfig `yaml:"qdrant"`
}

type MCPGitHubConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Toolsets []string `yaml:"toolsets"` // empty = server default toolset
}

type MCPQdrantConfig struct {
	Enabled bool   `yaml:"enabled"`
	URL     string `yaml:"url"` // e.g. "http://qdrant-mcp:3000/mcp"
}

type Neo4jConfig struct {
	Enabled bool   `yaml:"enabled"`
	URI     string `yaml:"uri"`
	User    string `yaml:"user"`
}

type AgentsConfig struct {
	Prompt string `yaml:"prompt"`
}

type TelegramConfig struct {
	Token     string   `yaml:"token"`
	Whitelist []string `yaml:"whitelist"`
	Password  string   `yaml:"password"`
}

// Load загружает конфигурацию из YAML и переменных окружения.
func Load(configPath string) (*Config, error) {
	envPath := ".env"

	// Загружаем .env (не критично, если отсутствует)
	if err := godotenv.Load(envPath); err != nil {
		fmt.Printf("Warning: .env file not found at %s, using system environment variables\n", envPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &Config{
		MaxReActSteps: 10,
		ScanInterval:  "24h",
		LogLevel:      "info",
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.overrideFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	return cfg, nil
}

func (c *Config) overrideFromEnv() error {
	if key := os.Getenv("LLM_API_KEY_OPENROUTER"); key != "" {
		c.LLM.OpenRouterAPIKey = key
	}

	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		c.Telegram.Token = token
	}

	if whitelist := os.Getenv("TELEGRAM_WHITELIST"); whitelist != "" {
		users := strings.Split(whitelist, ",")
		c.Telegram.Whitelist = make([]string, 0, len(users))
		for _, user := range users {
			if trimmed := strings.TrimSpace(user); trimmed != "" {
				c.Telegram.Whitelist = append(c.Telegram.Whitelist, trimmed)
			}
		}
	}

	if password := os.Getenv("TELEGRAM_PASSWORD"); password != "" {
		c.Telegram.Password = password
	}

	c.GitHubToken = os.Getenv("GITHUB_TOKEN")

	if neo4jPassword := os.Getenv("NEO4J_PASSWORD"); neo4jPassword != "" {
		c.Neo4jPassword = neo4jPassword
	}

	if qdrantMCPURL := os.Getenv("QDRANT_MCP_URL"); qdrantMCPURL != "" {
		c.MCP.Qdrant.URL = qdrantMCPURL
	}

	return nil
}

// MustLoad загружает конфигурацию или паникует (для использования в main).
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

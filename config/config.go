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
	MCPServers      []MCPServer    `yaml:"mcps"`
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

type MCPServer struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type AgentsConfig struct {
	Prompt string `yaml:"prompt"`
}

type TelegramConfig struct {
	Token     string   `yaml:"token"`
	Whitelist []string `yaml:"whitelist"`
	Password  string   `yaml:"password"`
	// ChatID полностью удалён — бот получает его динамически из входящих сообщений
}

// Load загружает конфигурацию из YAML и переменных окружения
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
	// LLM API ключ
	if key := os.Getenv("LLM_API_KEY_OPENROUTER"); key != "" {
		c.LLM.OpenRouterAPIKey = key
	}

	// Telegram
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

	return nil
}

// MustLoad загружает конфигурацию или паникует (для использования в main)
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// GetMCPServerURL возвращает URL для MCP сервера по его имени.
func (c *Config) GetMCPServerURL(name string) (string, error) {
	for _, srv := range c.MCPServers {
		if strings.EqualFold(srv.Name, name) {
			return srv.URL, nil
		}
	}
	return "", fmt.Errorf("MCP server %q not found", name)
}

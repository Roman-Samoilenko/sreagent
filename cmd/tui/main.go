package main

import (
	"fmt"
	"os"

	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/internal/app"
	"github.com/roman-samoilenko/sreagent/internal/tui"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
)

func main() {
	if len(os.Args) < 2 {
		os.Args = append(os.Args, "./config/config.yaml")
	}
	cfgPath := os.Args[1]

	cfg := config.MustLoad(cfgPath)

	// Открываем файл для логов
	logFile, err := os.OpenFile("logs/tui_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Инициализируем логгер с записью в файл
	logger.Init(cfg.LogLevel, logFile)

	application, err := app.New(cfg)
	if err != nil {
		logger.Error("Failed to create application", "error", err)
		fmt.Fprintf(os.Stderr, "Error creating app: %v\n", err)
		os.Exit(1)
	}
	defer application.Close()

	if application.MCPManager != nil {
		toolNames := application.MCPManager.ToolNames()
		logger.Info("Available MCP tools", "count", len(toolNames))
	}

	// Запускаем TUI
	if err := tui.Run(application.Orchestrator); err != nil {
		logger.Error("TUI error", "error", err)
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

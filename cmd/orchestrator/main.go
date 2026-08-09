package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/internal/app"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
)

func main() {
	cfg := config.MustLoad("config/config.yaml")

	// Инициализируем глобальный логгер
	logFile, err := os.OpenFile("logs/orchestrator_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Инициализируем глобальный логгер с выводом в файл
	logger.Init(cfg.LogLevel, logFile)

	application, err := app.New(cfg)
	if err != nil {
		logger.Fatal("Failed to create application", "error", err)
	}
	defer application.Close()

	logger.Info("Application started with MCP servers")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	<-ch

	logger.Info("Shutting down gracefully...")
}

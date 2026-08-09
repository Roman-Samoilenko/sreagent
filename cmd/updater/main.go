package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/internal/app"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to config file")
	flag.Parse()

	cfg := config.MustLoad(*configPath)
	logFile, err := os.OpenFile("logs/updater_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Cannot open log file: %v\n", err)
        os.Exit(1)
    }
    defer logFile.Close()

    // Инициализируем глобальный логгер с выводом в файл
    logger.Init(cfg.LogLevel, logFile)

	application, err := app.New(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize application", "error", err)
	}
	defer application.Close()

	scanInterval, err := time.ParseDuration(cfg.ScanInterval)
	if err != nil {
		logger.Fatal("Invalid scan_interval", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("Shutting down updater...")
		cancel()
	}()

	scanAll := func() {
		for _, repo := range cfg.Repositories {
			taskID := uuid.New().String()
			query := fmt.Sprintf(
				"Выполни полный анализ безопасности репозитория %s...", repo,
			)
			logger.Info("Scanning repository", "taskID", taskID, "repo", repo)
			result, err := application.Orchestrator.RunTask(ctx, taskID, query)
			if err != nil {
				logger.Error("Task failed", "taskID", taskID, "error", err)
				continue
			}
			logger.Info("Task completed", "taskID", taskID, "result_preview", result[:min(100, len(result))])
		}
	}

	scanAll()
	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			scanAll()
		case <-ctx.Done():
			return
		}
	}
}


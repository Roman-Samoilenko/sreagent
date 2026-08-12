package main

import (
	"fmt"
	"os"

	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/internal/bot"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
)

func main() {
	configPath := "config/config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}
	cfg := config.MustLoad(configPath)

	logFile, err := os.OpenFile("logs/bot_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	logger.Init(cfg.LogLevel, logFile)

	if err := bot.Start(cfg); err != nil {
		logger.Fatal("Bot failed", "error", err)
	}
}

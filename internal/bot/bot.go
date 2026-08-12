package bot

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/roman-samoilenko/sreagent/config"
	"github.com/roman-samoilenko/sreagent/internal/app"
	"github.com/roman-samoilenko/sreagent/internal/core"
	"github.com/roman-samoilenko/sreagent/pkg/logger"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	app    *app.App
	config *config.Config

	sessions sync.Map
}

func Start(cfg *config.Config) error {
	// Инициализируем приложение (LLM, инструменты)
	application, err := app.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create app: %w", err)
	}
	defer application.Close()

	if cfg.Telegram.Token == "" {
		return fmt.Errorf("telegram token is not set")
	}

	botAPI, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return fmt.Errorf("failed to create bot API: %w", err)
	}
	botAPI.Debug = true // true для отладки

	bot := &Bot{
		api:    botAPI,
		app:    application,
		config: cfg,
	}

	logger.Info("Telegram bot started", "username", botAPI.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := botAPI.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		go bot.handleMessage(update.Message)
	}
	return nil
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	userID := msg.From.ID

	// Проверка whitelist (если задан)
	if len(b.config.Telegram.Whitelist) > 0 {
		allowed := false
		for _, userName := range b.config.Telegram.Whitelist {
			if userName == msg.From.UserName || userName == fmt.Sprintf("%d", userID) {
				allowed = true
			}
		}
		if !allowed {
			b.reply(chatID, "Access denied. Your user ID is not whitelisted.")
			return
		}
	}

	// Получаем или создаём оркестратор для этого чата
	orch := b.getOrCreateOrchestrator(chatID)

	// Выполняем задачу
	ctx := context.Background()
	taskID := fmt.Sprintf("tg-%d-%d", chatID, msg.MessageID)
	query := msg.Text

	logger.Debug("Bot received message", "chatID", chatID, "query", query)

	result, err := orch.RunTask(ctx, taskID, query)
	if err != nil {
		logger.Error("Bot task failed", "chatID", chatID, "error", err)
		b.reply(chatID, "Error processing your request: "+err.Error())
		return
	}

	// Отправляем ответ (разбиваем, если слишком длинный)
	if len(result) > 4096 {
		parts := splitMessage(result, 4000)
		for _, part := range parts {
			b.reply(chatID, part)
		}
	} else {
		b.reply(chatID, result)
	}
}

func (b *Bot) reply(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML // или Markdown, но осторожно
	_, err := b.api.Send(msg)
	if err != nil {
		logger.Error("Failed to send message", "chatID", chatID, "error", err)
	}
}

func (b *Bot) getOrCreateOrchestrator(chatID int64) core.Orchestrator {
	val, ok := b.sessions.Load(chatID)
	if ok {
		return val.(core.Orchestrator)
	}
	orch, err := b.app.NewOrchestrator()
	if err != nil {
		logger.Error("Failed to create orchestrator for chat", "chatID", chatID, "error", err)
		return nil
	}
	b.sessions.Store(chatID, orch)
	return orch
}

// splitMessage разбивает длинный текст на части, не разрывая строки.
func splitMessage(text string, maxLen int) []string {
	var parts []string
	for len(text) > maxLen {
		idx := strings.LastIndex(text[:maxLen], "\n")
		if idx == -1 {
			idx = maxLen
		}
		parts = append(parts, text[:idx])
		text = text[idx:]
	}
	if text != "" {
		parts = append(parts, text)
	}
	return parts
}

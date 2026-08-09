package logger

import (
	"io"
	"log/slog"
	"os"
)

var globalLogger *Logger

// Init настраивает глобальный логгер с указанным уровнем.
// Должна вызываться один раз при старте приложения.
func Init(level string, output io.Writer) {
	globalLogger = New(level, output)
}

func New(level string, output io.Writer) *Logger {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelDebug
	}
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{
		Level: lvl,
	})
	return &Logger{slog.New(handler)}
}

type Logger struct {
	*slog.Logger
}

func Debug(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Error(msg, args...)
	}
}

// Fatal логирует сообщение с уровнем Error и завершает программу с кодом 1.
func Fatal(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Error(msg, args...)
	}
	os.Exit(1)
}

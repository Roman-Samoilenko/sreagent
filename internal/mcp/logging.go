package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/roman-samoilenko/sreagent/pkg/logger"
)

// LoggingManager оборачивает обычный Manager и добавляет логирование
type LoggingManager struct {
	inner *Manager
	mu    sync.RWMutex
	calls []CallLog
}

// CallLog представляет запись о вызове инструмента
type CallLog struct {
	Timestamp time.Time
	ToolName  string
	Arguments map[string]interface{}
	Result    interface{}
	Error     error
	Duration  time.Duration
}

// NewLoggingManager создаёт менеджер с логированием
func NewLoggingManager(inner *Manager) *LoggingManager {
	return &LoggingManager{
		inner: inner,
		calls: make([]CallLog, 0),
	}
}

// RegisterServer регистрирует сервер
func (m *LoggingManager) RegisterServer(server Server) error {
	return m.inner.RegisterServer(server)
}

// InitializeAll инициализирует все серверы
func (m *LoggingManager) InitializeAll(ctx context.Context) error {
	logger.Info("Initializing MCP servers...")
	start := time.Now()

	if err := m.inner.InitializeAll(ctx); err != nil {
		logger.Error("Failed to initialize MCP servers", "error", err, "duration", time.Since(start))
		return err
	}

	logger.Info("MCP servers initialized successfully", "duration", time.Since(start))
	return nil
}

// LoadToolsFromServers загружает инструменты
func (m *LoggingManager) LoadToolsFromServers(ctx context.Context) error {
	logger.Info("Loading tools from MCP servers...")
	start := time.Now()

	if err := m.inner.LoadToolsFromServers(ctx); err != nil {
		logger.Error("Failed to load tools from servers", "error", err, "duration", time.Since(start))
		return err
	}

	toolCount := len(m.inner.tools)
	logger.Info("Tools loaded successfully", "count", toolCount, "duration", time.Since(start))

	for fullName := range m.inner.tools {
		logger.Debug("Tool loaded", "name", fullName)
	}

	return nil
}

// GetToolNames возвращает список инструментов
func (m *LoggingManager) GetToolNames(ctx context.Context) []string {
	return m.inner.GetToolNames(ctx)
}

// GetToolMetadata возвращает метаданные инструмента
func (m *LoggingManager) GetToolMetadata(toolFullName string) (*ToolHandle, error) {
	return m.inner.GetToolMetadata(toolFullName)
}

// CallTool вызывает инструмент с логированием
func (m *LoggingManager) CallTool(ctx context.Context, toolFullName string, arguments map[string]interface{}) (interface{}, error) {
	start := time.Now()

	// Логируем вызов
	argsJSON, _ := json.Marshal(arguments)
	logger.Debug("MCP tool call started",
		"tool", toolFullName,
		"json.Marshal(arguments).(string)", string(argsJSON),
		"arguments", arguments,
	)

	// Выполняем вызов
	result, err := m.inner.CallTool(ctx, toolFullName, arguments)
	duration := time.Since(start)

	// Логируем результат
	if err != nil {
		logger.Error("MCP tool call failed",
			"tool", toolFullName,
			"error", err.Error(),
			"duration", duration.String(),
		)
	} else {
		resultJSON, _ := json.Marshal(result)
		logger.Debug("MCP tool call succeeded",
			"tool", toolFullName,
			"result_preview", limitString(string(resultJSON), 200),
			"duration", duration.String(),
		)
	}

	// Сохраняем запись о вызове
	m.mu.Lock()
	m.calls = append(m.calls, CallLog{
		Timestamp: start,
		ToolName:  toolFullName,
		Arguments: arguments,
		Result:    result,
		Error:     err,
		Duration:  duration,
	})
	// Храним только последние 1000 вызовов
	if len(m.calls) > 1000 {
		m.calls = m.calls[len(m.calls)-1000:]
	}
	m.mu.Unlock()

	return result, err
}

// GetCallLogs возвращает историю вызовов
func (m *LoggingManager) GetCallLogs() []CallLog {
	m.mu.RLock()
	defer m.mu.RUnlock()

	logs := make([]CallLog, len(m.calls))
	copy(logs, m.calls)
	return logs
}

// Close закрывает менеджер
func (m *LoggingManager) Close() error {
	return m.inner.Close()
}

// RecoveryManager оборачивает менеджер и добавляет обработку паник
type RecoveryManager struct {
	inner *LoggingManager
}

// NewRecoveryManager создаёт менеджер с обработкой ошибок
func NewRecoveryManager(inner *LoggingManager) *RecoveryManager {
	return &RecoveryManager{inner: inner}
}

// CallToolWithRecovery вызывает инструмент с обработкой паник
func (m *RecoveryManager) CallToolWithRecovery(ctx context.Context, toolFullName string, arguments map[string]interface{}) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("MCP tool panicked",
				"tool", toolFullName,
				"panic", fmt.Sprintf("%v", r),
			)
			err = fmt.Errorf("tool %s panicked: %v", toolFullName, r)
		}
	}()

	return m.inner.CallTool(ctx, toolFullName, arguments)
}

// limitString ограничивает длину строки
func limitString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ErrorHandler обрабатывает ошибки от MCP
type ErrorHandler struct {
	maxRetries int
	retryDelay time.Duration
}

// NewErrorHandler создаёт обработчик ошибок
func NewErrorHandler(maxRetries int, retryDelay time.Duration) *ErrorHandler {
	return &ErrorHandler{
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// CallWithRetry вызывает функцию с повторами при ошибке
func (h *ErrorHandler) CallWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying after error", "attempt", attempt, "delay", h.retryDelay)
			select {
			case <-time.After(h.retryDelay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err
		logger.Warn("Attempt failed", "attempt", attempt+1, "error", err)
	}

	return fmt.Errorf("failed after %d attempts: %w", h.maxRetries+1, lastErr)
}

// ValidateToolCall проверяет валидность вызова инструмента
func ValidateToolCall(manager *Manager, toolName string, arguments map[string]interface{}) error {
	handle, err := manager.GetToolMetadata(toolName)
	if err != nil {
		return fmt.Errorf("tool %q not found", toolName)
	}

	// Проверяем требуемые аргументы из schema
	if schema, ok := handle.Metadata.InputSchema["required"]; ok {
		if required, ok := schema.([]interface{}); ok {
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := arguments[reqStr]; !exists {
						return fmt.Errorf("missing required argument: %q", reqStr)
					}
				}
			}
		}
	}

	return nil
}

// MetricsCollector собирает метрики использования инструментов
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*ToolMetrics
}

// ToolMetrics представляет метрики инструмента
type ToolMetrics struct {
	Name         string
	CallCount    int
	SuccessCount int
	FailureCount int
	TotalTime    time.Duration
	AvgTime      time.Duration
	LastError    string
}

// NewMetricsCollector создаёт сборщик метрик
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*ToolMetrics),
	}
}

// RecordCall записывает вызов инструмента
func (mc *MetricsCollector) RecordCall(toolName string, duration time.Duration, err error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.metrics[toolName]; !exists {
		mc.metrics[toolName] = &ToolMetrics{Name: toolName}
	}

	m := mc.metrics[toolName]
	m.CallCount++
	m.TotalTime += duration
	m.AvgTime = m.TotalTime / time.Duration(m.CallCount)

	if err != nil {
		m.FailureCount++
		m.LastError = err.Error()
	} else {
		m.SuccessCount++
	}
}

// GetMetrics возвращает метрики для инструмента
func (mc *MetricsCollector) GetMetrics(toolName string) *ToolMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if m, exists := mc.metrics[toolName]; exists {
		return m
	}
	return nil
}

// GetAllMetrics возвращает все метрики
func (mc *MetricsCollector) GetAllMetrics() map[string]*ToolMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]*ToolMetrics)
	for k, v := range mc.metrics {
		result[k] = v
	}
	return result
}

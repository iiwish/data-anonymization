package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// LogLevel 日志级别
type LogLevel string

const (
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

// LogEntry 表示一条日志记录
type LogEntry struct {
	Timestamp    string   `json:"timestamp"`
	RequestID    string   `json:"request_id"`
	Service      string   `json:"service"`
	SystemID     string   `json:"system_id,omitempty"`
	UserID       string   `json:"user_id,omitempty"`
	Status       string   `json:"status"`
	LatencyMs    int64    `json:"latency_ms,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
	Level        LogLevel `json:"level"`
}

// Logger 日志记录器
type Logger struct {
	file *os.File
	mu   sync.Mutex
}

var globalLogger *Logger

// Init 初始化全局日志记录器
func Init(logFilePath string) error {
	// 创建日志目录
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	// 打开或创建日志文件（追加模式）
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	globalLogger = &Logger{
		file: file,
	}

	return nil
}

// Close 关闭日志记录器
func Close() error {
	if globalLogger != nil && globalLogger.file != nil {
		return globalLogger.file.Close()
	}
	return nil
}

// LogRequest 记录请求日志
func LogRequest(service, systemID, userID, status string, latencyMs int64, errMsg string) {
	if globalLogger == nil {
		return
	}

	entry := LogEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		RequestID:    uuid.New().String(),
		Service:      service,
		SystemID:     systemID,
		UserID:       userID,
		Status:       status,
		LatencyMs:    latencyMs,
		ErrorMessage: errMsg,
		Level:        LevelInfo,
	}

	if status != "SUCCESS" {
		entry.Level = LevelError
	}

	globalLogger.write(entry)
}

// LogRequestWithID 记录带有自定义请求ID的日志
func LogRequestWithID(requestID, service, systemID, userID, status string, latencyMs int64, errMsg string) {
	if globalLogger == nil {
		return
	}

	entry := LogEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		RequestID:    requestID,
		Service:      service,
		SystemID:     systemID,
		UserID:       userID,
		Status:       status,
		LatencyMs:    latencyMs,
		ErrorMessage: errMsg,
		Level:        LevelInfo,
	}

	if status != "SUCCESS" {
		entry.Level = LevelError
	}

	globalLogger.write(entry)
}

// Info 记录信息级别日志
func Info(service, message string) {
	if globalLogger == nil {
		return
	}

	entry := LogEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		RequestID:    uuid.New().String(),
		Service:      service,
		Status:       "INFO",
		ErrorMessage: message,
		Level:        LevelInfo,
	}

	globalLogger.write(entry)
}

// Error 记录错误级别日志
func Error(service, message string) {
	if globalLogger == nil {
		return
	}

	entry := LogEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		RequestID:    uuid.New().String(),
		Service:      service,
		Status:       "ERROR",
		ErrorMessage: message,
		Level:        LevelError,
	}

	globalLogger.write(entry)
}

// write 写入日志条目
func (l *Logger) write(entry LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	l.file.Write(data)
	l.file.WriteString("\n")
}

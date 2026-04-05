package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// LogLevel 表示a log level。
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)

// LogEntry 表示a log entry。
type LogEntry struct {
	Level     LogLevel       `json:"level"`
	Tag       string         `json:"tag"`
	Message   string         `json:"message"`
	Timestamp string         `json:"timestamp"`
	Ctx       map[string]any `json:"ctx,omitempty"`
}

// LogSink 表示a log sink。
type LogSink interface {
	Write(entry LogEntry)
}

// Logger 表示a logger。
type Logger interface {
	Debug(msg string, ctx ...map[string]any)
	Info(msg string, ctx ...map[string]any)
	Warn(msg string, ctx ...map[string]any)
	Error(msg string, ctx ...map[string]any)
	Child(tag string, ctx ...map[string]any) Logger
}

// StderrSink 表示a stderr log sink。
type StderrSink struct {
	MinLevel     LogLevel
	EnableColors bool
}

// NewStderrSink 创建新的stderr sink。
func NewStderrSink(options ...func(*StderrSink)) *StderrSink {
	sink := &StderrSink{
		MinLevel:     InfoLevel,
		EnableColors: false, // Will be set based on environment
	}
	for _, opt := range options {
		opt(sink)
	}
	return sink
}

// Write 写入a log entry to stderr。
func (s *StderrSink) Write(entry LogEntry) {
	if levelOrder(entry.Level) < levelOrder(s.MinLevel) {
		return
	}

	levelTag := fmt.Sprintf("%-5s", entry.Level.LevelString())
	prefix := fmt.Sprintf("[%s]", entry.Tag)

	if s.EnableColors {
		color := levelColor(entry.Level)
		reset := "\033[0m"
		fmt.Fprintf(os.Stderr, "%s%s%s %s %s\n", color, levelTag, reset, prefix, entry.Message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s %s\n", levelTag, prefix, entry.Message)
	}
}

// LevelString 返回the uppercase level string。
func (l LogLevel) LevelString() string {
	return string(l)
}

// JSONLineSink 表示a JSON line log sink。
type JSONLineSink struct {
	Writer interface{ Write([]byte) (int, error) }
}

// Write 写入a log entry as JSON。
func (s *JSONLineSink) Write(entry LogEntry) {
	if s.Writer == nil {
		return
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	_, _ = s.Writer.Write(append(data, '\n'))
}

// NullSink 表示a null log sink。
type NullSink struct{}

// Write discards a log entry
func (s *NullSink) Write(entry LogEntry) {}

// loggerImpl implements the Logger interface
type loggerImpl struct {
	tag      string
	sinks    []LogSink
	minLevel LogLevel
	baseCtx  map[string]any
}

// NewLogger 创建新的logger。
func NewLogger(tag string, sinks []LogSink, minLevel LogLevel, baseCtx ...map[string]any) Logger {
	ctx := make(map[string]any)
	for _, c := range baseCtx {
		for k, v := range c {
			ctx[k] = v
		}
	}
	return &loggerImpl{
		tag:      tag,
		sinks:    sinks,
		minLevel: minLevel,
		baseCtx:  ctx,
	}
}

func (l *loggerImpl) Debug(msg string, ctx ...map[string]any) {
	l.emit(DebugLevel, msg, ctx...)
}

func (l *loggerImpl) Info(msg string, ctx ...map[string]any) {
	l.emit(InfoLevel, msg, ctx...)
}

func (l *loggerImpl) Warn(msg string, ctx ...map[string]any) {
	l.emit(WarnLevel, msg, ctx...)
}

func (l *loggerImpl) Error(msg string, ctx ...map[string]any) {
	l.emit(ErrorLevel, msg, ctx...)
}

func (l *loggerImpl) Child(tag string, ctx ...map[string]any) Logger {
	newCtx := make(map[string]any)
	for k, v := range l.baseCtx {
		newCtx[k] = v
	}
	for _, c := range ctx {
		for k, v := range c {
			newCtx[k] = v
		}
	}
	return &loggerImpl{
		tag:      tag,
		sinks:    l.sinks,
		minLevel: l.minLevel,
		baseCtx:  newCtx,
	}
}

func (l *loggerImpl) emit(level LogLevel, msg string, ctx ...map[string]any) {
	if levelOrder(level) < levelOrder(l.minLevel) {
		return
	}

	entry := LogEntry{
		Level:     level,
		Tag:       l.tag,
		Message:   msg,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if len(l.baseCtx) > 0 || len(ctx) > 0 {
		entry.Ctx = make(map[string]any)
		for k, v := range l.baseCtx {
			entry.Ctx[k] = v
		}
		for _, c := range ctx {
			for k, v := range c {
				entry.Ctx[k] = v
			}
		}
	}

	for _, sink := range l.sinks {
		sink.Write(entry)
	}
}

func levelOrder(level LogLevel) int {
	switch level {
	case DebugLevel:
		return 0
	case InfoLevel:
		return 1
	case WarnLevel:
		return 2
	case ErrorLevel:
		return 3
	default:
		return 1
	}
}

func levelColor(level LogLevel) string {
	switch level {
	case DebugLevel:
		return "\033[90m" // gray
	case InfoLevel:
		return "\033[36m" // cyan
	case WarnLevel:
		return "\033[33m" // yellow
	case ErrorLevel:
		return "\033[31m" // red
	default:
		return "\033[0m"
	}
}

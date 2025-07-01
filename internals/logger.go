package internals

import (
	"fmt"
	"log"
	"time"
)

const (
	LogLevelDebug = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

const (
	EnvProd = "PROD"
	EnvDev  = "DEV"
)

type Logger struct {
	level int
	env   string
}

func NewLogger(env string) *Logger {
	var level int
	switch env {
	case EnvProd:
		level = LogLevelInfo
	default:
		level = LogLevelDebug
	}

	return &Logger{
		level: level,
		env:   env,
	}
}

func (l *Logger) Log(level int, format string, args ...any) {
	if level < l.level {
		return
	}

	prefix := [...]string{"DEBUG", "INFO", "WARNING", "ERROR"}[level]
	timestamp := time.Now().Format(time.RFC3339)
	msg := fmt.Sprintf(format, args...)
	log.Printf("[%s] %s %s\n", prefix, timestamp, msg)
}

func (l *Logger) Debug(format string, args ...any) {
	l.Log(LogLevelDebug, format, args...)
}

func (l *Logger) Info(format string, args ...any) {
	l.Log(LogLevelInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...any) {
	l.Log(LogLevelWarn, format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.Log(LogLevelError, format, args...)
}

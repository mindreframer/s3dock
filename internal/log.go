package internal

import (
	"fmt"
	"os"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelError LogLevel = 1
	LogLevelInfo  LogLevel = 2
	LogLevelDebug LogLevel = 3
)

// Logger interface for structured logging
type Logger interface {
	Error(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
	SetLevel(level LogLevel)
	GetLevel() LogLevel
}

// logger implements the Logger interface
type logger struct {
	level LogLevel
}

// Global logger instance
var globalLogger Logger = &logger{level: LogLevelInfo} // Default to info level

// GetLogger returns the global logger instance
func GetLogger() Logger {
	return globalLogger
}

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(l Logger) {
	globalLogger = l
}

// SetLevel sets the log level for the global logger
func SetLogLevel(level LogLevel) {
	if l, ok := globalLogger.(*logger); ok {
		l.SetLevel(level)
	}
}

// Error logs error messages (level 1+)
func (l *logger) Error(msg string, args ...interface{}) {
	if l.level >= LogLevelError {
		l.log("ERROR", msg, args...)
	}
}

// Info logs info messages (level 2+)
func (l *logger) Info(msg string, args ...interface{}) {
	if l.level >= LogLevelInfo {
		l.log("INFO", msg, args...)
	}
}

// Debug logs debug messages (level 3+)
func (l *logger) Debug(msg string, args ...interface{}) {
	if l.level >= LogLevelDebug {
		l.log("DEBUG", msg, args...)
	}
}

// SetLevel sets the log level
func (l *logger) SetLevel(level LogLevel) {
	l.level = level
}

// GetLevel returns the current log level
func (l *logger) GetLevel() LogLevel {
	return l.level
}

// log formats and outputs the log message
func (l *logger) log(level, msg string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var formattedMsg string
	if len(args) > 0 {
		formattedMsg = fmt.Sprintf(msg, args...)
	} else {
		formattedMsg = msg
	}

	logMsg := fmt.Sprintf("[%s] %s %s", level, timestamp, formattedMsg)
	fmt.Fprintln(os.Stderr, logMsg)
}

// Convenience functions for global logger
func LogError(msg string, args ...interface{}) {
	globalLogger.Error(msg, args...)
}

func LogInfo(msg string, args ...interface{}) {
	globalLogger.Info(msg, args...)
}

func LogDebug(msg string, args ...interface{}) {
	globalLogger.Debug(msg, args...)
}

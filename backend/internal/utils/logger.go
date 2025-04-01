package utils

import (
	"fmt"
	"os"

	"github.com/digital-egiz/backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger with additional methods
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new logger instance with the given configuration
func NewLogger(config *config.LogConfig) (*Logger, error) {
	var zapConfig zap.Config

	// Configure logger based on format
	if config.Format == "json" {
		zapConfig = zap.NewProductionConfig()
	} else {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Configure log level
	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	// Configure output
	if config.OutputPath != "stdout" && config.OutputPath != "stderr" {
		zapConfig.OutputPaths = []string{config.OutputPath}
		zapConfig.ErrorOutputPaths = []string{config.OutputPath}
	} else if config.OutputPath == "stderr" {
		zapConfig.OutputPaths = []string{"stderr"}
		zapConfig.ErrorOutputPaths = []string{"stderr"}
	} else {
		// Default to stdout
		zapConfig.OutputPaths = []string{"stdout"}
		zapConfig.ErrorOutputPaths = []string{"stdout"}
	}

	// Configure encoding
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create logger
	zapLogger, err := zapConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Sync logger on program exit
	zap.ReplaceGlobals(zapLogger)
	zap.RedirectStdLog(zapLogger)

	return &Logger{
		Logger: zapLogger,
	}, nil
}

// Named returns a new logger with the given name added to the logger's name
func (l *Logger) Named(name string) *Logger {
	return &Logger{
		Logger: l.Logger.Named(name),
	}
}

// With returns a new logger with the given fields added to the logger's context
func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
	}
}

// Debug logs a message at Debug level
func (l *Logger) Debug(msg string, fields ...zapcore.Field) {
	l.Logger.Debug(msg, fields...)
}

// Info logs a message at Info level
func (l *Logger) Info(msg string, fields ...zapcore.Field) {
	l.Logger.Info(msg, fields...)
}

// Warn logs a message at Warn level
func (l *Logger) Warn(msg string, fields ...zapcore.Field) {
	l.Logger.Warn(msg, fields...)
}

// Error logs a message at Error level
func (l *Logger) Error(msg string, fields ...zapcore.Field) {
	l.Logger.Error(msg, fields...)
}

// Fatal logs a message at Fatal level and then calls os.Exit(1)
func (l *Logger) Fatal(msg string, fields ...zapcore.Field) {
	l.Logger.Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// Close syncs the logger and handles any cleanup
func (l *Logger) Close() error {
	if err := l.Sync(); err != nil {
		// Ignore sync errors as they can occur when stdout/stderr is closed
		if err != os.ErrInvalid {
			return err
		}
	}
	return nil
}

// Field aliases for convenience
var (
	String  = zap.String
	Int     = zap.Int
	Bool    = zap.Bool
	Float64 = zap.Float64
	Error   = zap.Error
	Any     = zap.Any
)

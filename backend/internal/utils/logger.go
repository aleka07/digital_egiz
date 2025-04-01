package utils

import (
	"github.com/digital-egiz/backend/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is a wrapper around zap.Logger
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new logger with the given configuration
func NewLogger(cfg *config.LoggingConfig) (*Logger, error) {
	var zapCfg zap.Config

	if cfg.Production {
		zapCfg = zap.NewProductionConfig()
	} else {
		zapCfg = zap.NewDevelopmentConfig()
	}

	// Set log level
	switch cfg.Level {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		zapCfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{logger}, nil
}

// With adds structured context to the logger
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{l.Logger.With(fields...)}
}

// Named adds a sub-logger with the specified name
func (l *Logger) Named(name string) *Logger {
	return &Logger{l.Logger.Named(name)}
}

// Sugar returns a sugared logger
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.Logger.Sugar()
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
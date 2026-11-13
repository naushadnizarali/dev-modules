// Package logger builds configured Zap loggers for applications that want
// human-readable console logs, JSON file logs, or both.
package logger

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config controls the logger outputs and encoding.
type Config struct {
	Level         string
	Console       bool
	File          bool
	FilePath      string
	Development   bool
	Caller        bool
	Stacktrace    bool
	ConsoleWriter io.Writer
}

// DefaultConfig returns a production-ready console configuration.
func DefaultConfig() Config {
	return Config{
		Level:       "info",
		Console:     true,
		Caller:      true,
		Stacktrace:  true,
		Development: false,
	}
}

// New creates a logger from cfg. Console output is human-readable and file
// output is newline-delimited JSON. Call Sync before the application exits.
func New(cfg Config) (*zap.Logger, error) {
	if !cfg.Console && !cfg.File {
		return nil, errors.New("logger: enable console, file, or both")
	}

	level := zapcore.InfoLevel
	if value := strings.TrimSpace(cfg.Level); value != "" {
		if err := level.UnmarshalText([]byte(value)); err != nil {
			return nil, fmt.Errorf("logger: invalid level %q: %w", cfg.Level, err)
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	if cfg.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	cores := make([]zapcore.Core, 0, 2)
	if cfg.Console {
		writer := cfg.ConsoleWriter
		if writer == nil {
			writer = os.Stdout
		}
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(writer),
			level,
		))
	}

	if cfg.File {
		path := strings.TrimSpace(cfg.FilePath)
		if path == "" {
			return nil, errors.New("logger: file path is required when file output is enabled")
		}
		if dir := filepath.Dir(path); dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("logger: create log directory: %w", err)
			}
		}
		file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("logger: open log file: %w", err)
		}
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(file),
			level,
		))
	}

	options := make([]zap.Option, 0, 3)
	if cfg.Caller {
		options = append(options, zap.AddCaller())
	}
	if cfg.Development {
		options = append(options, zap.Development())
	}
	if cfg.Stacktrace {
		stackLevel := zapcore.ErrorLevel
		if cfg.Development {
			stackLevel = zapcore.WarnLevel
		}
		options = append(options, zap.AddStacktrace(stackLevel))
	}

	return zap.New(zapcore.NewTee(cores...), options...), nil
}

// MustNew is like New but panics when the configuration is invalid.
func MustNew(cfg Config) *zap.Logger {
	log, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return log
}

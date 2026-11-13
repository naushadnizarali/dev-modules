package logger

import (
	"errors"
	"sync/atomic"

	"go.uber.org/zap"
)

// Logger is the formatted, application-wide logging interface exposed by Log.
// Its methods use fmt.Printf-style formatting.
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Fatal(format string, args ...any)
	Sync() error
}

// Log is the application-wide formatted logger. Before Init or SetGlobal is
// called it safely discards entries.
var Log Logger = globalLog

var globalLog = newGlobalLogger()

type globalLogger struct {
	logger atomic.Pointer[zap.Logger]
}

func newGlobalLogger() *globalLogger {
	log := &globalLogger{}
	log.logger.Store(zap.NewNop())
	return log
}

// Init creates a logger from cfg and installs it as both Log and zap.L(). Call
// Sync from main during application shutdown.
func Init(cfg Config) error {
	log, err := New(cfg)
	if err != nil {
		return err
	}
	return SetGlobal(log)
}

// SetGlobal installs an existing Zap logger as both Log and zap.L(). This is
// useful when the application needs to add Zap options before installation.
func SetGlobal(log *zap.Logger) error {
	if log == nil {
		return errors.New("logger: global logger cannot be nil")
	}
	globalLog.logger.Store(log)
	zap.ReplaceGlobals(log)
	return nil
}

// Sync flushes buffered entries from the application-wide logger.
func Sync() error {
	return globalLog.Sync()
}

// Debug writes a formatted debug entry to Log.
func Debug(format string, args ...any) {
	Log.Debug(format, args...)
}

// Info writes a formatted informational entry to Log.
func Info(format string, args ...any) {
	Log.Info(format, args...)
}

// Warn writes a formatted warning entry to Log.
func Warn(format string, args ...any) {
	Log.Warn(format, args...)
}

// Error writes a formatted error entry to Log.
func Error(format string, args ...any) {
	Log.Error(format, args...)
}

// Fatal writes a formatted fatal entry and terminates the process.
func Fatal(format string, args ...any) {
	Log.Fatal(format, args...)
}

func (l *globalLogger) current() *zap.SugaredLogger {
	return l.logger.Load().Sugar()
}

func (l *globalLogger) Debug(format string, args ...any) {
	l.current().Debugf(format, args...)
}

func (l *globalLogger) Info(format string, args ...any) {
	l.current().Infof(format, args...)
}

func (l *globalLogger) Warn(format string, args ...any) {
	l.current().Warnf(format, args...)
}

func (l *globalLogger) Error(format string, args ...any) {
	l.current().Errorf(format, args...)
}

func (l *globalLogger) Fatal(format string, args ...any) {
	l.current().Fatalf(format, args...)
}

func (l *globalLogger) Sync() error {
	return l.logger.Load().Sync()
}

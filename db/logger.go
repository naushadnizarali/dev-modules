package db

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type zapGORMLogger struct {
	log            *zap.Logger
	level          gormlogger.LogLevel
	slowThreshold  time.Duration
	ignoreNotFound bool
}

func (l *zapGORMLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := *l
	clone.level = level
	return &clone
}

func (l *zapGORMLogger) Info(_ context.Context, message string, args ...any) {
	if l.level >= gormlogger.Info {
		l.log.Sugar().Infof(message, args...)
	}
}

func (l *zapGORMLogger) Warn(_ context.Context, message string, args ...any) {
	if l.level >= gormlogger.Warn {
		l.log.Sugar().Warnf(message, args...)
	}
}

func (l *zapGORMLogger) Error(_ context.Context, message string, args ...any) {
	if l.level >= gormlogger.Error {
		l.log.Sugar().Errorf(message, args...)
	}
}

func (l *zapGORMLogger) Trace(_ context.Context, begin time.Time, query func() (string, int64), err error) {
	if l.level == gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	isError := err != nil && l.level >= gormlogger.Error && !(l.ignoreNotFound && errors.Is(err, gorm.ErrRecordNotFound))
	isSlow := l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= gormlogger.Warn
	isInfo := l.level >= gormlogger.Info
	if !isError && !isSlow && !isInfo {
		return
	}

	sql, rows := query()
	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	switch {
	case isError:
		l.log.Error("gorm query failed", append(fields, zap.Error(err))...)
	case isSlow:
		l.log.Warn("gorm slow query", append(fields, zap.Duration("threshold", l.slowThreshold))...)
	case isInfo:
		l.log.Info("gorm query", fields...)
	}
}

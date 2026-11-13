// Package db opens configured GORM database connections.
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	DriverPostgres  = "postgres"
	DriverMySQL     = "mysql"
	DriverSQLite    = "sqlite"
	DriverSQLServer = "sqlserver"
)

// Config controls the database driver, DSN, GORM behavior, and connection
// pool. Pool values are applied only when greater than zero.
type Config struct {
	Env             string // "production", "staging", "development", etc.
	Driver          string
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	PrepareStmt     bool
	TranslateError  bool
	Logger          *zap.Logger
	LogLevel        string
	SlowThreshold   time.Duration
	IgnoreNotFound  bool
}

// DefaultConfig returns conservative connection-pool defaults. Driver and DSN
// must be supplied by the application.
func DefaultConfig() Config {
	return Config{
		MaxIdleConns:    10,
		MaxOpenConns:    100,
		ConnMaxIdleTime: 10 * time.Minute,
		ConnMaxLifetime: time.Hour,
		LogLevel:        "warn",
		SlowThreshold:   200 * time.Millisecond,
		IgnoreNotFound:  true,
	}
}

// Open creates and verifies a GORM connection. The caller should invoke Close
// during application shutdown.
func Open(ctx context.Context, cfg Config) (*gorm.DB, error) {
	if ctx == nil {
		return nil, errors.New("db: context is required")
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, errors.New("db: DSN is required")
	}
	if cfg.MaxIdleConns < 0 || cfg.MaxOpenConns < 0 || cfg.ConnMaxIdleTime < 0 || cfg.ConnMaxLifetime < 0 {
		return nil, errors.New("db: connection pool values cannot be negative")
	}
	if cfg.MaxOpenConns > 0 && cfg.MaxIdleConns > cfg.MaxOpenConns {
		return nil, errors.New("db: MaxIdleConns cannot exceed MaxOpenConns")
	}

	dialector, err := dialector(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, err
	}

	log, err := newGORMLogger(cfg)
	if err != nil {
		return nil, err
	}

	database, err := gorm.Open(dialector, &gorm.Config{
		DisableAutomaticPing: true,
		PrepareStmt:          cfg.PrepareStmt,
		TranslateError:       cfg.TranslateError,
		Logger:               log,
	})
	if err != nil {
		return nil, fmt.Errorf("db: open %s connection: %w", normalizeDriver(cfg.Driver), err)
	}

	if cfg.Env != "production" {
		database = database.Debug()
		zap.L().Info("🔍 DB Debug mode enabled (non-production)")
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("db: access connection pool: %w", err)
	}

	configurePool(sqlDB, cfg)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("db: ping %s connection: %w", normalizeDriver(cfg.Driver), err)
	}

	return database, nil
}

func newGORMLogger(cfg Config) (gormlogger.Interface, error) {
	if cfg.Logger == nil {
		return gormlogger.Discard, nil
	}
	if cfg.SlowThreshold < 0 {
		return nil, errors.New("db: SlowThreshold cannot be negative")
	}

	level, err := parseLogLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	return &zapGORMLogger{
		log:            cfg.Logger,
		level:          level,
		slowThreshold:  cfg.SlowThreshold,
		ignoreNotFound: cfg.IgnoreNotFound,
	}, nil
}

func parseLogLevel(value string) (gormlogger.LogLevel, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "warn", "warning":
		return gormlogger.Warn, nil
	case "silent", "off", "none":
		return gormlogger.Silent, nil
	case "error":
		return gormlogger.Error, nil
	case "info", "debug":
		return gormlogger.Info, nil
	default:
		return 0, fmt.Errorf("db: invalid GORM log level %q (supported: silent, error, warn, info)", value)
	}
}

// Close closes the underlying sql.DB connection pool. It is safe to call with
// a nil GORM database.
func Close(database *gorm.DB) error {
	if database == nil {
		return nil
	}
	sqlDB, err := database.DB()
	if err != nil {
		return fmt.Errorf("db: access connection pool: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("db: close connection pool: %w", err)
	}
	return nil
}

func dialector(driver, dsn string) (gorm.Dialector, error) {
	switch normalizeDriver(driver) {
	case DriverPostgres:
		return postgres.Open(dsn), nil
	case DriverMySQL:
		return mysql.Open(dsn), nil
	case DriverSQLite:
		return sqlite.Open(dsn), nil
	case DriverSQLServer:
		return sqlserver.Open(dsn), nil
	default:
		return nil, fmt.Errorf(
			"db: unsupported driver %q (supported: postgres, mysql, sqlite, sqlserver)",
			driver,
		)
	}
}

func normalizeDriver(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "postgres", "postgresql", "pgx":
		return DriverPostgres
	case "mysql", "mariadb":
		return DriverMySQL
	case "sqlite", "sqlite3":
		return DriverSQLite
	case "sqlserver", "mssql":
		return DriverSQLServer
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func configurePool(sqlDB *sql.DB, cfg Config) {
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
}

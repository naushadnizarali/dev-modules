// Package app initializes shared application configuration and services.
package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/naushadnizarali/dev-modules/database"
	"github.com/naushadnizarali/dev-modules/logger"
	moduleredis "github.com/naushadnizarali/dev-modules/redis"
	"github.com/naushadnizarali/dev-modules/utils"
	redisclient "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Config contains the environment configuration used during initialization.
// It contains credentials and should not be logged or exposed in HTTP responses.
type Config struct {
	Env         string
	HTTPAddr    string
	CorsOrigin  []string
	AppTimezone string
	AppName     string
	ViteApiURL  string
	ViteEnv     string
	Production  bool
	GinMode     string
	JWTSecret   string
	JWTIssuer   string
	JWTAudience []string
	JWTTTL      time.Duration
	Logger      logger.Config
	Database    database.Config
	Redis       moduleredis.Config
}

// App owns the initialized services. Call Close during application shutdown.
type App struct {
	Config    Config
	Redis     *redisclient.Client
	DB        *gorm.DB
	Logger    *zap.Logger
	closeOnce sync.Once
	closeErr  error
}

func parseConfig() (Config, error) {
	cfg := Config{
		Env:         utils.GetEnv("APP_ENV", "development"),
		Production:  utils.GetBool("PRODUCTION", false),
		GinMode:     utils.GetEnv("GIN_MODE", "debug"),
		HTTPAddr:    utils.GetEnv("HTTP_ADDR", ":8080"),
		CorsOrigin:  utils.SplitCSV(utils.GetEnv("CORS_ORIGIN", "*")),
		AppTimezone: utils.GetEnv("APP_TIMEZONE", "UTC"),
		AppName:     utils.GetEnv("APP_NAME", "dev-modules"),
		ViteApiURL:  utils.GetEnv("VITE_API_URL", "http://localhost:8080"),
		ViteEnv:     utils.GetEnv("VITE_ENV", "development"),
		JWTSecret:   utils.GetEnv("JWT_SECRET", "dev-modules-secret"),
		JWTIssuer:   utils.GetEnv("JWT_ISSUER", "dev-modules"),
		JWTAudience: utils.SplitCSV(utils.GetEnv("JWT_AUDIENCE", "dev-modules-api")),
		JWTTTL:      utils.GetDuration("JWT_TTL", 15*time.Minute),
	}
	if !strings.Contains(cfg.HTTPAddr, ":") {
		cfg.HTTPAddr = ":" + cfg.HTTPAddr
	}
	cfg.Logger = logger.DefaultConfig()
	cfg.Logger.Level = utils.GetEnv("LOG_LEVEL", "info")
	cfg.Logger.FilePath = utils.GetEnv("LOG_FILE", "")
	cfg.Logger.File = cfg.Logger.FilePath != ""

	cfg.Database = database.DefaultConfig()
	cfg.Database.Production = cfg.Production
	cfg.Database.Driver = utils.GetEnv("DATABASE_DRIVER", database.DriverSQLite)
	cfg.Database.DSN = utils.GetEnv("DATABASE_DSN", "")
	cfg.Database.Schema = utils.GetEnv("DATABASE_SCHEMA", "")
	cfg.Database.DoSeed = utils.GetBool("DATABASE_SEED", false)
	cfg.Database.LogLevel = utils.GetEnv("DATABASE_LOG_LEVEL", "warn")

	cfg.Redis = moduleredis.DefaultConfig()
	cfg.Redis.ClientName = strings.ToLower(utils.GetEnv("REDIS_CLIENT_NAME", "dev-modules"))
	cfg.Redis.Addr = utils.GetEnv("REDIS_ADDR", cfg.Redis.Addr)
	cfg.Redis.Username = utils.GetEnv("REDIS_USERNAME", "")
	cfg.Redis.Password = utils.GetEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = utils.GetInt("REDIS_DB", 0)
	cfg.Redis.RateLimitEnabled = utils.GetBool("REDIS_RATE_LIMIT_ENABLED", false)
	cfg.Redis.RateLimitCount = utils.GetInt64("REDIS_RATE_LIMIT_COUNT", 100)
	endpoint := utils.GetEnv("REDIS_URL", "")
	if endpoint != "" && !strings.Contains(endpoint, "://") {
		cfg.Redis.Addr = endpoint
	} else {
		cfg.Redis.URL = endpoint
	}
	return cfg, nil
}

// Init loads an optional .env file from the working directory (existing
// environment variables take precedence), then opens and verifies both services.
// It installs the logger for logger.Log and zap.L(). It does not start a server
// or require JWT configuration. The caller controls cancellation through ctx.
func Init(ctx context.Context) (*App, error) {
	if ctx == nil {
		return nil, errors.New("app: context is required")
	}
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("app: load .env: %w", err)
	}
	cfg, err := parseConfig()
	if err != nil {
		return nil, fmt.Errorf("app: parse config: %w", err)
	}

	err = logger.Init(cfg.Logger)
	if err != nil {
		return nil, fmt.Errorf("app: create logger: %w", err)
	}
	log := zap.L()
	cfg.Database.Logger = log
	result := &App{Config: cfg, Logger: log}
	startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	result.DB, err = database.Open(startupCtx, cfg.Database)
	if err != nil {
		_ = result.Close()
		return nil, fmt.Errorf("app: open database: %w", err)
	}
	result.Redis, err = moduleredis.Open(startupCtx, cfg.Redis)
	if err != nil {
		_ = result.Close()
		return nil, fmt.Errorf("app: open Redis: %w", err)
	}
	return result, nil
}

// Close closes Redis and the database and flushes the logger. Repeated calls
// return the same result. Stop handlers and workers before calling Close.
func (a *App) Close() error {
	if a == nil {
		return nil
	}
	a.closeOnce.Do(func() {
		var errs []error
		if a.Redis != nil {
			errs = append(errs, a.Redis.Close())
		}
		errs = append(errs, database.Close(a.DB))
		if a.Logger != nil {
			_ = a.Logger.Sync()
		} // Console streams may not support sync.
		a.closeErr = errors.Join(errs...)
	})
	return a.closeErr
}

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/naushadnizarali/dev-modules/auth"
	db "github.com/naushadnizarali/dev-modules/database"
	"github.com/naushadnizarali/dev-modules/logger"
	"github.com/naushadnizarali/dev-modules/middleware"
	moduleredis "github.com/naushadnizarali/dev-modules/redis"
	"github.com/naushadnizarali/dev-modules/utils"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ Could not load .env file")
		panic(err)
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	logConfig := logger.DefaultConfig()
	logConfig.Level = utils.GetEnv("LOG_LEVEL", "info")
	logConfig.FilePath = utils.GetEnv("LOG_FILE", "")
	logConfig.File = logConfig.FilePath != ""
	if err := logger.Init(logConfig); err != nil {
		return err
	}
	defer func() { _ = logger.Sync() }()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jwtManager, err := auth.New(auth.Config{
		Secret:   []byte(os.Getenv("JWT_SECRET")),
		Issuer:   "dev-modules",
		Audience: []string{"dev-modules-api"},
		TTL:      15 * time.Minute,
	})
	if err != nil {
		logger.Error("configure JWT: %v", err)
		return fmt.Errorf("configure JWT: %w", err)
	}

	startupCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	databaseConfig := db.DefaultConfig()
	databaseConfig.Driver = utils.GetEnv("DATABASE_DRIVER", db.DriverPostgres)
	databaseConfig.DSN = utils.GetEnv("DATABASE_DSN", "")
	databaseConfig.Logger = zap.L()
	database, err := db.Open(startupCtx, databaseConfig)
	if err != nil {
		logger.Error("open database: %v", err)
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { _ = db.Close(database) }()

	redisConfig := moduleredis.DefaultConfig()
	redisConfig.Addr = utils.GetEnv("REDIS_ADDR", redisConfig.Addr)
	redisEndpoint := utils.GetEnv("REDIS_URL", "")
	if redisEndpoint != "" && !strings.Contains(redisEndpoint, "://") {
		redisConfig.Addr = redisEndpoint
	} else {
		redisConfig.URL = redisEndpoint
	}
	redisClient, err := moduleredis.Open(startupCtx, redisConfig)
	if err != nil {
		logger.Error("open Redis: %v", err)
		return fmt.Errorf("open Redis: %w", err)
	}
	defer func() { _ = redisClient.Close() }()
	cancel()

	skipHealth := func(r *http.Request) bool { return r.URL.Path == "/health" }
	authenticate, err := middleware.Authenticate(middleware.AuthConfig{
		Manager: jwtManager,
		Skip:    skipHealth,
	})
	if err != nil {
		logger.Error("configure authentication: %v", err)
		return err
	}
	rateLimit, err := middleware.RateLimit(middleware.RateLimitConfig{
		Redis:  redisClient,
		Limit:  100,
		Window: time.Minute,
		Prefix: "dev-modules",
		Skip:   skipHealth,
		Key: func(r *http.Request) string {
			subject, _ := auth.StringClaim(r.Context(), "sub")
			return subject
		},
	})
	if err != nil {
		logger.Error("configure rate limiting: %v", err)
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /me", func(w http.ResponseWriter, r *http.Request) {
		subject, _ := auth.StringClaim(r.Context(), "sub")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(subject))
	})

	handler := authenticate(rateLimit(mux))
	handler = middleware.RequestLog(middleware.RequestLogConfig{Logger: zap.L()})(handler)
	httpAddr := utils.GetEnv("HTTP_ADDR", ":8080")
	if !strings.Contains(httpAddr, ":") {
		httpAddr = ":" + httpAddr
	}
	server := &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	logger.Info("HTTP server starting on %s", server.Addr)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve HTTP: %v", err)
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
		logger.Info("HTTP server shutting down")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		logger.Error("shutdown HTTP: %v", err)
		return fmt.Errorf("shutdown HTTP: %w", err)
	}
	return nil
}

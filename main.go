package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/naushadnizarali/dev-modules/app"
	"github.com/naushadnizarali/dev-modules/auth"
	"github.com/naushadnizarali/dev-modules/logger"
	"github.com/naushadnizarali/dev-modules/middleware"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	services, err := app.AppInit(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = services.Close() }()

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
		Redis:  services.Redis,
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
	handler = middleware.RequestLog(middleware.RequestLogConfig{Logger: services.Logger})(handler)
	server := &http.Server{
		Addr:              services.Config.HTTPAddr,
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

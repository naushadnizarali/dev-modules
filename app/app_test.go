package app_test

import (
	"context"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/naushadnizarali/dev-modules/app"
	"github.com/naushadnizarali/dev-modules/logger"
	"go.uber.org/zap"
)

func TestAppInitAndClose(t *testing.T) {
	t.Chdir(t.TempDir())
	server := miniredis.RunT(t)
	t.Setenv("DATABASE_DRIVER", "sqlite")
	t.Setenv("DATABASE_DSN", ":memory:")
	t.Setenv("REDIS_URL", server.Addr())
	t.Setenv("HTTP_ADDR", "4441")
	t.Setenv("LOG_FILE", "")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("APP_ENV", "production")
	t.Setenv("PRODUCTION", "true")
	t.Setenv("JWT_SECRET", "") // Shared services do not require JWT setup.
	t.Cleanup(func() { _ = logger.SetGlobal(zap.NewNop()) })
	services, err := app.Init(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = services.Close() })
	if services.Config.HTTPAddr != ":4441" || services.Config.Database.Production != true {
		t.Fatalf("unexpected configuration")
	}
	if services.Logger != zap.L() {
		t.Fatal("global logger not installed")
	}
	if err := services.DB.Exec("CREATE TABLE example (id INTEGER)").Error; err != nil {
		t.Fatal(err)
	}
	if err := services.Redis.Set(context.Background(), "example", "ok", 0).Err(); err != nil {
		t.Fatal(err)
	}
	if got, err := server.Get("example"); err != nil || got != "ok" {
		t.Fatalf("Redis value = %q, %v", got, err)
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
	if err := services.Close(); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := services.DB.DB()
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlDB.Ping(); err == nil {
		t.Fatal("database is still open")
	}
	if err := services.Redis.Ping(context.Background()).Err(); err == nil {
		t.Fatal("Redis is still open")
	}
}

func TestAppInitFailureInitializesGlobalLogger(t *testing.T) {
	t.Cleanup(func() { _ = logger.SetGlobal(zap.NewNop()) })
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_DRIVER", "sqlite")
	t.Setenv("DATABASE_DSN", ":memory:")
	t.Setenv("REDIS_URL", "invalid://endpoint")
	t.Setenv("LOG_FILE", "")
	t.Setenv("LOG_LEVEL", "error")
	original := zap.NewNop()
	if err := logger.SetGlobal(original); err != nil {
		t.Fatal(err)
	}
	services, err := app.Init(context.Background())
	if err == nil || services != nil || !strings.Contains(err.Error(), "open Redis") {
		t.Fatalf("AppInit = %v, %v", services, err)
	}
	if zap.L() == original {
		t.Fatal("global logger was not initialized before opening connections")
	}
}

func TestAppInitRejectsNilContext(t *testing.T) {
	if _, err := app.Init(nil); err == nil {
		t.Fatal("expected error")
	}
}

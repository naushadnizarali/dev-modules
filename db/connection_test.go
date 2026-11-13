package db_test

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/naushadnizarali/dev-modules/db"
	modulelogger "github.com/naushadnizarali/dev-modules/logger"
)

type record struct {
	ID   uint
	Name string
}

func TestOpenSQLiteAndConfigurePool(t *testing.T) {
	cfg := db.DefaultConfig()
	cfg.Driver = "sqlite3"
	cfg.DSN = filepath.Join(t.TempDir(), "app.db")
	cfg.MaxIdleConns = 1
	cfg.MaxOpenConns = 1

	database, err := db.Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(database); err != nil {
			t.Error(err)
		}
	})

	if err := database.AutoMigrate(&record{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&record{Name: "ready"}).Error; err != nil {
		t.Fatal(err)
	}
	var got record
	if err := database.First(&got).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != "ready" {
		t.Fatalf("unexpected record: %#v", got)
	}

	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	if got := sqlDB.Stats().MaxOpenConnections; got != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", got)
	}
}

func TestOpenValidatesConfig(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		cfg  db.Config
	}{
		{name: "nil context", cfg: db.Config{Driver: db.DriverSQLite, DSN: ":memory:"}},
		{name: "missing DSN", ctx: context.Background(), cfg: db.Config{Driver: db.DriverSQLite}},
		{name: "unknown driver", ctx: context.Background(), cfg: db.Config{Driver: "oracle", DSN: "dsn"}},
		{name: "negative pool value", ctx: context.Background(), cfg: db.Config{Driver: db.DriverSQLite, DSN: ":memory:", MaxOpenConns: -1}},
		{name: "idle exceeds open", ctx: context.Background(), cfg: db.Config{Driver: db.DriverSQLite, DSN: ":memory:", MaxIdleConns: 2, MaxOpenConns: 1}},
		{name: "invalid log level", ctx: context.Background(), cfg: db.Config{Driver: db.DriverSQLite, DSN: ":memory:", Logger: modulelogger.MustNew(modulelogger.DefaultConfig()), LogLevel: "trace"}},
		{name: "negative slow threshold", ctx: context.Background(), cfg: db.Config{Driver: db.DriverSQLite, DSN: ":memory:", Logger: modulelogger.MustNew(modulelogger.DefaultConfig()), SlowThreshold: -1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := db.Open(test.ctx, test.cfg); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestGORMLoggerUsesConfiguredZapOutput(t *testing.T) {
	var output bytes.Buffer
	log, err := modulelogger.New(modulelogger.Config{
		Level:         "info",
		Console:       true,
		ConsoleWriter: &output,
	})
	if err != nil {
		t.Fatal(err)
	}

	cfg := db.Config{
		Driver:       db.DriverSQLite,
		DSN:          filepath.Join(t.TempDir(), "logged.db"),
		Logger:       log,
		LogLevel:     "info",
		MaxOpenConns: 1,
	}
	database, err := db.Open(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close(database) })

	if err := database.Exec("SELECT 1").Error; err != nil {
		t.Fatal(err)
	}
	if err := log.Sync(); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, "gorm query") || !strings.Contains(got, "SELECT 1") {
		t.Fatalf("unexpected GORM log output: %q", got)
	}
}

func TestCloseNil(t *testing.T) {
	if err := db.Close(nil); err != nil {
		t.Fatal(err)
	}
}

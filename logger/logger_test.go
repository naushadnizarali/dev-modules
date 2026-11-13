package logger_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/naushadnizarali/dev-modules/logger"
	"go.uber.org/zap"
)

func TestNewWritesConsoleAndFileAtConfiguredLevel(t *testing.T) {
	var console bytes.Buffer
	path := filepath.Join(t.TempDir(), "nested", "app.log")
	log, err := logger.New(logger.Config{
		Level:         "warn",
		Console:       true,
		File:          true,
		FilePath:      path,
		ConsoleWriter: &console,
	})
	if err != nil {
		t.Fatal(err)
	}

	log.Info("hidden")
	log.Warn("visible", zap.String("component", "test"))
	if err := log.Sync(); err != nil {
		t.Fatal(err)
	}

	if got := console.String(); !strings.Contains(got, "visible") || strings.Contains(got, "hidden") {
		t.Fatalf("unexpected console output: %q", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entry map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &entry); err != nil {
		t.Fatalf("file output is not JSON: %v", err)
	}
	if entry["msg"] != "visible" || entry["component"] != "test" {
		t.Fatalf("unexpected file entry: %#v", entry)
	}
}

func TestNewValidatesConfiguration(t *testing.T) {
	tests := []struct {
		name string
		cfg  logger.Config
	}{
		{name: "no output"},
		{name: "missing file path", cfg: logger.Config{File: true}},
		{name: "invalid level", cfg: logger.Config{Console: true, Level: "verbose"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := logger.New(test.cfg); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

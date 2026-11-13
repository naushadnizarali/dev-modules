package logger_test

import (
	"bytes"
	"strings"
	"testing"

	modulelogger "github.com/naushadnizarali/dev-modules/logger"
	"go.uber.org/zap"
)

func TestGlobalLogger(t *testing.T) {
	t.Cleanup(func() { _ = modulelogger.SetGlobal(zap.NewNop()) })

	var output bytes.Buffer
	err := modulelogger.Init(modulelogger.Config{
		Level:         "debug",
		Console:       true,
		ConsoleWriter: &output,
	})
	if err != nil {
		t.Fatal(err)
	}

	modulelogger.Info("started on port %d", 8080)
	modulelogger.Debug("component=%s", "api")
	if err := modulelogger.Sync(); err != nil {
		t.Fatal(err)
	}

	got := output.String()
	for _, value := range []string{"started on port 8080", "component=api"} {
		if !strings.Contains(got, value) {
			t.Fatalf("output %q does not contain %q", got, value)
		}
	}
	if zap.L().Core().Enabled(zap.DebugLevel) != true {
		t.Fatal("Init did not install the logger as zap.L()")
	}
}

func TestSetGlobalRejectsNil(t *testing.T) {
	if err := modulelogger.SetGlobal(nil); err == nil {
		t.Fatal("expected nil logger error")
	}
}

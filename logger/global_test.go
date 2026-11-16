package logger_test

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	modulelogger "github.com/naushadnizarali/dev-modules/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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

func TestGlobalLoggerCaller(t *testing.T) {
	t.Cleanup(func() { _ = modulelogger.SetGlobal(zap.NewNop()) })
	core, entries := observer.New(zap.DebugLevel)
	if err := modulelogger.SetGlobal(zap.New(core, zap.AddCaller())); err != nil {
		t.Fatal(err)
	}

	for _, call := range []func(string, ...any){
		modulelogger.Debug, modulelogger.Info, modulelogger.Warn, modulelogger.Error,
		modulelogger.Log.Debug, modulelogger.Log.Info, modulelogger.Log.Warn, modulelogger.Log.Error,
	} {
		_, file, line, _ := runtime.Caller(0)
		call("caller test")
		entry := entries.All()[entries.Len()-1]
		if entry.Caller.File != file || entry.Caller.Line != line+1 {
			t.Errorf("caller = %s, want %s:%d", entry.Caller, file, line+1)
		}
	}

	_, file, line, _ := runtime.Caller(0)
	zap.L().Info("direct Zap caller")
	entry := entries.All()[entries.Len()-1]
	if entry.Caller.File != file || entry.Caller.Line != line+1 {
		t.Errorf("Zap caller = %s, want %s:%d", entry.Caller, file, line+1)
	}
}

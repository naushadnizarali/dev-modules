package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	modulelogger "github.com/naushadnizarali/dev-modules/logger"
	"github.com/naushadnizarali/dev-modules/middleware"
)

func TestRequestLogRecordsRequestAndResponse(t *testing.T) {
	var output bytes.Buffer
	log, err := modulelogger.New(modulelogger.Config{
		Level:         "info",
		Console:       true,
		ConsoleWriter: &output,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := middleware.RequestLog(middleware.RequestLogConfig{Logger: log})(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("created"))
		}),
	)
	request := httptest.NewRequest(http.MethodPost, "/widgets?active=true", nil)
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	got := output.String()
	for _, value := range []string{"http request completed", "POST", "/widgets", "request-123", "201", "7"} {
		if !strings.Contains(got, value) {
			t.Fatalf("log output %q does not contain %q", got, value)
		}
	}
}

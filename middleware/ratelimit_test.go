package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/naushadnizarali/dev-modules/middleware"
	"github.com/redis/go-redis/v9"
)

func TestRateLimit(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr(), Protocol: 2})
	t.Cleanup(func() { _ = client.Close() })

	wrap, err := middleware.RateLimit(middleware.RateLimitConfig{
		Redis:  client,
		Limit:  2,
		Window: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	for requestNumber, wantStatus := range []int{http.StatusNoContent, http.StatusNoContent, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodGet, "/resource", nil)
		request.RemoteAddr = "192.0.2.1:4321"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != wantStatus {
			t.Fatalf("request %d status = %d, want %d", requestNumber+1, response.Code, wantStatus)
		}
		if response.Header().Get("X-RateLimit-Limit") != "2" {
			t.Fatalf("request %d missing rate limit headers", requestNumber+1)
		}
	}
}

func TestRateLimitValidatesConfig(t *testing.T) {
	if _, err := middleware.RateLimit(middleware.RateLimitConfig{}); err == nil {
		t.Fatal("expected configuration error")
	}
}

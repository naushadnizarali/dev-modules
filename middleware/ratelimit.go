// Package middleware provides reusable net/http middleware.
package middleware

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var rateLimitScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
local ttl = redis.call('PTTL', KEYS[1])
return {count, ttl}
`)

// RateLimitConfig controls a fixed-window, Redis-backed rate limiter.
type RateLimitConfig struct {
	Redis    *redis.Client
	Limit    int64
	Window   time.Duration
	Prefix   string
	Key      func(*http.Request) string
	Skip     func(*http.Request) bool
	FailOpen bool
}

// RateLimit returns middleware that shares counters across application
// instances through Redis.
func RateLimit(cfg RateLimitConfig) (func(http.Handler) http.Handler, error) {
	if cfg.Redis == nil {
		return nil, errors.New("middleware: rate limit Redis client is required")
	}
	if cfg.Limit <= 0 {
		return nil, errors.New("middleware: rate limit must be greater than zero")
	}
	if cfg.Window <= 0 {
		return nil, errors.New("middleware: rate limit window must be greater than zero")
	}
	if cfg.Prefix == "" {
		cfg.Prefix = "rate_limit"
	}
	if cfg.Key == nil {
		cfg.Key = clientIP
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			identity := strings.TrimSpace(cfg.Key(r))
			if identity == "" {
				identity = "unknown"
			}
			key := fmt.Sprintf("%s:%s", cfg.Prefix, identity)
			result, err := rateLimitScript.Run(r.Context(), cfg.Redis, []string{key}, cfg.Window.Milliseconds()).Int64Slice()
			if err != nil {
				if cfg.FailOpen {
					next.ServeHTTP(w, r)
					return
				}
				http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
				return
			}

			count, ttl := result[0], time.Duration(result[1])*time.Millisecond
			remaining := max(cfg.Limit-count, 0)
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(cfg.Limit, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(max(ttl, 0)).Unix(), 10))
			if count > cfg.Limit {
				retryAfter := max(int64((ttl+time.Second-1)/time.Second), 1)
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

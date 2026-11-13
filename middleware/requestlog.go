package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// RequestLogConfig controls HTTP request and response metadata logging.
type RequestLogConfig struct {
	Logger          *zap.Logger
	RequestIDHeader string
	Skip            func(*http.Request) bool
}

// RequestLog logs one structured entry after each response. Response bodies
// and sensitive request headers are deliberately not logged.
func RequestLog(cfg RequestLogConfig) func(http.Handler) http.Handler {
	log := cfg.Logger
	if log == nil {
		log = zap.NewNop()
	}
	if cfg.RequestIDHeader == "" {
		cfg.RequestIDHeader = "X-Request-ID"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.Skip != nil && cfg.Skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			started := time.Now()
			response := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
			defer func() {
				recovered := recover()
				if recovered != nil && !response.wrote {
					response.status = http.StatusInternalServerError
				}
				fields := []zap.Field{
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("query", r.URL.RawQuery),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.UserAgent()),
					zap.String("request_id", r.Header.Get(cfg.RequestIDHeader)),
					zap.Int("status", response.status),
					zap.Int("response_bytes", response.bytes),
					zap.Duration("duration", time.Since(started)),
				}
				if recovered != nil {
					log.Error("http request panicked", append(fields, zap.Any("panic", recovered))...)
					panic(recovered)
				}
				log.Info("http request completed", fields...)
			}()
			next.ServeHTTP(response, r)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (w *responseRecorder) WriteHeader(status int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseRecorder) Write(data []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach optional interfaces implemented by
// the original response writer.
func (w *responseRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

package httpx

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// statusRecorder wraps http.ResponseWriter so that loggingMiddleware can
// observe the status code that downstream handlers wrote. It writes the
// captured value through to the underlying ResponseWriter unchanged.
//
// Hijack passthrough: gorilla/websocket's Upgrade requires the
// ResponseWriter to satisfy http.Hijacker so it can take over the
// underlying TCP connection. Without an explicit forwarder, the
// embedding here hides the inner Hijack and the upgrade fails with 500.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader captures the status code, then forwards.
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Hijack delegates to the underlying ResponseWriter when supported.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("ResponseWriter does not support Hijack")
	}
	return hj.Hijack()
}

// Flush passes through to the underlying ResponseWriter when supported.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// loggingMiddleware emits a single slog INFO record per request with
// method, path, status, and duration_ms. Query values, headers, and
// body bytes are intentionally never logged so tokens / role secrets
// cannot leak through the access log (NFR-U4-S2).
func loggingMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

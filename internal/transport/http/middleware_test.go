package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStatusRecorder_Captures(t *testing.T) {
	w := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	rec.WriteHeader(http.StatusTeapot)
	if rec.status != http.StatusTeapot {
		t.Errorf("status = %d", rec.status)
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("forwarded status = %d", w.Code)
	}
}

func TestLoggingMiddleware_LogsFourFields(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	handler := loggingMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	r := httptest.NewRequest("GET", "/foo?secret=token", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status passed through = %d", w.Code)
	}
	out := buf.String()
	for _, want := range []string{"method=GET", "path=/foo", "status=201", "duration_ms="} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %q: %s", want, out)
		}
	}
	// Critical: query value must NOT be logged.
	if strings.Contains(out, "token") || strings.Contains(out, "secret") {
		t.Errorf("query value leaked into log: %s", out)
	}
}

func TestLoggingMiddleware_DefaultStatusIs200(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	handler := loggingMiddleware(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	r := httptest.NewRequest("GET", "/foo", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !strings.Contains(buf.String(), "status=200") {
		t.Errorf("default status not 200: %s", buf.String())
	}
}

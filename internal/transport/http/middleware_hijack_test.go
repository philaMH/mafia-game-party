package httpx

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// hijackable simulates a real http.ResponseWriter that supports Hijack
// (httptest.ResponseRecorder does not).
type hijackable struct {
	http.ResponseWriter
	hijacked bool
	flushed  bool
}

func (h *hijackable) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.hijacked = true
	return nil, nil, nil
}

func (h *hijackable) Flush() { h.flushed = true }

func TestStatusRecorder_HijackForwards(t *testing.T) {
	inner := &hijackable{ResponseWriter: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: inner, status: 200}
	if _, _, err := rec.Hijack(); err != nil {
		t.Fatalf("Hijack: %v", err)
	}
	if !inner.hijacked {
		t.Error("inner Hijack not invoked")
	}
}

func TestStatusRecorder_HijackUnsupported(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: 200}
	if _, _, err := rec.Hijack(); err == nil {
		t.Error("expected error when underlying writer is not Hijacker")
	}
}

func TestStatusRecorder_FlushForwards(t *testing.T) {
	inner := &hijackable{ResponseWriter: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: inner, status: 200}
	rec.Flush()
	if !inner.flushed {
		t.Error("inner Flush not invoked")
	}
}

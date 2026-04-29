package httpx

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintLANAddresses_OutputsHTTPLines(t *testing.T) {
	var buf bytes.Buffer
	PrintLANAddresses(&buf, 8080)
	out := buf.String()

	// Either we got at least one private LAN address OR the localhost
	// fallback. Both are acceptable on test runners.
	if !strings.Contains(out, ":8080") {
		t.Errorf("expected port :8080 in output: %q", out)
	}
	if !strings.HasPrefix(strings.TrimLeft(out, " "), "http://") {
		t.Errorf("expected http:// prefix: %q", out)
	}
}

func TestPrintLANAddresses_FallbackOnEmpty(t *testing.T) {
	var buf bytes.Buffer
	PrintLANAddresses(&buf, 9999)
	if !strings.Contains(buf.String(), ":9999") {
		t.Errorf("expected port :9999, got %q", buf.String())
	}
}

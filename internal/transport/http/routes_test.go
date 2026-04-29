package httpx

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHealthHandler(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/healthz", nil)
	healthHandler(w, r)
	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("body = %q", w.Body.String())
	}
}

func TestSPAHandler_MissingIndexReturns503(t *testing.T) {
	emptyFS := fstest.MapFS{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	spaHandler(emptyFS).ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d", w.Code)
	}
}

func TestSPAHandler_ServesIndex(t *testing.T) {
	assets := fstest.MapFS{"index.html": {Data: []byte("<html>spa</html>")}}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/play", nil)
	spaHandler(assets).ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "spa") {
		t.Errorf("body = %q", w.Body.String())
	}
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("cache = %q", cc)
	}
}

func TestAssetsHandler_SetsImmutableHeader(t *testing.T) {
	assets := fstest.MapFS{
		"assets/main-abc.js": {Data: []byte("console.log(\"x\")")},
	}
	h := assetsHandler(assets)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/assets/main-abc.js", nil)
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Errorf("status = %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("cache header missing immutable: %q", w.Header().Get("Cache-Control"))
	}
}

func TestAssetsHandler_404OnMissing(t *testing.T) {
	assets := fstest.MapFS{}
	h := assetsHandler(assets)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/assets/nope.js", nil)
	h.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Errorf("status = %d", w.Code)
	}
}

func TestBuildMux_RegistersAllRoutes(t *testing.T) {
	cfg := Config{
		Hub:   dummyHub{},
		Store: dummyStore{},
		Assets: fstest.MapFS{
			"index.html":         {Data: []byte("ok")},
			"assets/main-abc.js": {Data: []byte("hi")},
		},
	}
	mux := buildMux(cfg, nil)
	cases := map[string]int{
		"/healthz":            200, // healthHandler
		"/api/results":        200, // dummyStore → empty results, encoded as JSON
		"/assets/main-abc.js": 200, // file exists in MapFS
		"/play":               200, // SPA fallback → index.html
		"/ws":                 200, // dummyHub.UpgradeHandler is a no-op (200 default)
	}
	for path, want := range cases {
		t.Run(path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", path, nil)
			mux.ServeHTTP(w, r)
			if w.Code != want {
				t.Errorf("route %s status = %d, want %d", path, w.Code, want)
			}
		})
	}
}

// Compile-time guard: testAssets returns the right interface.
var _ fs.FS = testAssets()

package httpx

import (
	"io"
	"io/fs"
	"log/slog"
	"net/http"
)

// buildMux registers every route owned by the HTTP layer onto a fresh
// ServeMux and returns it. Pattern order matters only for documentation
// — ServeMux selects the most-specific match at runtime (Go 1.22+).
func buildMux(cfg Config, log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthHandler)
	mux.Handle("GET /ws", cfg.Hub.UpgradeHandler())
	mux.Handle("GET /api/results", resultsHandler(cfg.Store, log))
	mux.Handle("GET /assets/", assetsHandler(cfg.Assets))
	// Iter7 — pre-recorded host narration. Files are served by audioId
	// (e.g. /audio/phase.night.mp3); when missing, the host client
	// gracefully skips per FR-8.8.
	mux.Handle("GET /audio/", audioHandler(cfg.Assets))
	// Catch-all SPA fallback. ServeMux will route specific patterns
	// above first; everything else lands here and receives index.html
	// so React Router can resolve the path client-side.
	mux.Handle("GET /", spaHandler(cfg.Assets))

	return mux
}

// healthHandler answers GET /healthz with a plain "ok" so external
// liveness probes (e.g., a launcher script) can confirm the server is
// up without parsing JSON.
func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// assetsHandler serves Vite's hashed bundle directory under
// `/assets/`. Filenames carry a content hash so we can mark them
// immutable; clients then bypass revalidation on every reload.
func assetsHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})
}

// audioHandler serves pre-recorded narration MP3s under `/audio/`.
// Unlike `/assets/`, audio filenames are not content-hashed (they map
// to stable audioId values), so we use a short cache window — operators
// can replace an mp3 by dropping a new file and rebuilding without
// fighting client caches. When a file is missing the FileServer returns
// 404 and the host client falls through to subtitle-only display per
// Iter7 FR-8.8.
func audioHandler(assets fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(assets))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		fileServer.ServeHTTP(w, r)
	})
}

// spaHandler is the catch-all that returns the SPA index.html. It is
// the only HTML response and uses no-cache so the browser always picks
// up the freshest bundle reference. /api/* and /assets/* never reach
// here because they are routed by more specific ServeMux patterns.
func spaHandler(assets fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := assets.Open("index.html")
		if err != nil {
			http.Error(w, "frontend not built", http.StatusServiceUnavailable)
			return
		}
		defer func() { _ = f.Close() }()

		info, err := f.Stat()
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		rs, ok := f.(io.ReadSeeker)
		if !ok {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, "index.html", info.ModTime(), rs)
	})
}

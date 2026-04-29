package httpx

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/transport/ws"
)

// Server is the HTTP layer's public lifecycle interface. It owns an
// underlying *http.Server but hides its surface so callers cannot
// accidentally bypass middleware or timeouts.
type Server interface {
	// ListenAndServe blocks until the server stops. It returns
	// http.ErrServerClosed on a graceful Shutdown — the caller should
	// treat that as success.
	ListenAndServe() error

	// Shutdown initiates a graceful shutdown. The provided ctx bounds
	// the wait for in-flight requests (NFR-U4-R1: < 5s).
	Shutdown(ctx context.Context) error
}

// Config carries the dependencies required by the HTTP server. All
// fields except Logger are required; New returns an error otherwise.
type Config struct {
	Addr   string
	Hub    ws.Hub
	Store  persistence.PersistenceStore
	Assets fs.FS
	Logger *slog.Logger
}

// server is the default Server implementation.
type server struct {
	cfg     Config
	httpSrv *http.Server
	log     *slog.Logger
}

// New constructs a Server with the routes and middleware described in
// the U4 NFR Design (P-U4-1, P-U4-2, P-U4-3, P-U4-6). It does not
// bind the port; ListenAndServe must be called explicitly.
func New(cfg Config) (Server, error) {
	if cfg.Addr == "" {
		return nil, errors.New("httpx: missing Config.Addr")
	}
	if cfg.Hub == nil {
		return nil, errors.New("httpx: missing Config.Hub")
	}
	if cfg.Store == nil {
		return nil, errors.New("httpx: missing Config.Store")
	}
	if cfg.Assets == nil {
		return nil, errors.New("httpx: missing Config.Assets")
	}

	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	mux := buildMux(cfg, log)
	handler := loggingMiddleware(log)(mux)

	return &server{
		cfg: cfg,
		log: log,
		httpSrv: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second, // slowloris guard
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      0, // 0 = unlimited so WS upgrade survives long-lived connections
			IdleTimeout:       60 * time.Second,
		},
	}, nil
}

// ListenAndServe implements Server.
func (s *server) ListenAndServe() error {
	return s.httpSrv.ListenAndServe()
}

// Shutdown implements Server.
func (s *server) Shutdown(ctx context.Context) error {
	return s.httpSrv.Shutdown(ctx)
}

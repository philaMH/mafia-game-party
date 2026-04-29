// Command mafia-game is the LAN host binary for the mafia game PoC. It
// wires every internal unit together (game engine, session manager,
// announce catalog, persistence store, WebSocket hub, HTTP server),
// prints reachable LAN URLs, and handles graceful shutdown on
// SIGINT/SIGTERM.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
	"github.com/saltware/mafia-game/internal/session"
	httpx "github.com/saltware/mafia-game/internal/transport/http"
	"github.com/saltware/mafia-game/internal/transport/ws"
)

//go:embed all:web/dist
var webDist embed.FS

func main() {
	port := flag.Int("port", envInt("MAFIA_PORT", 8080), "HTTP listen port")
	dbPath := flag.String("db", envStr("MAFIA_DB_PATH", "./data/mafia.db"), "SQLite database file path")
	logLevel := flag.String("log-level", envStr("MAFIA_LOG_LEVEL", "info"), "log level (debug|info|warn|error)")
	flag.Parse()

	if *port < 1 || *port > 65535 {
		fmt.Fprintf(os.Stderr, "invalid port %d\n", *port)
		os.Exit(2)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLevel(*logLevel)}))
	slog.SetDefault(log)

	if err := run(*port, *dbPath, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(port int, dbPath string, log *slog.Logger) error {
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := persistence.OpenSqlite(rootCtx, dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}

	engine := game.NewDefault(game.NewDefaultKeywordPool())
	catalog := announce.NewDefaultCatalog()

	mgr, err := session.New(store, catalog, engine, nil, nil, session.SessionOpts{})
	if err != nil {
		return fmt.Errorf("session.New: %w", err)
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true }, // LAN-only deployment
	}
	hub := ws.New(upgrader, mgr, log)

	assets, err := fs.Sub(webDist, "web/dist")
	if err != nil {
		return fmt.Errorf("embed sub: %w", err)
	}

	srv, err := httpx.New(httpx.Config{
		Addr:   fmt.Sprintf("0.0.0.0:%d", port),
		Hub:    hub,
		Store:  store,
		Assets: assets,
		Logger: log,
	})
	if err != nil {
		return fmt.Errorf("httpx.New: %w", err)
	}

	fmt.Printf("mafia-game listening on:\n")
	httpx.PrintLANAddresses(os.Stdout, port)

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("listen: %w", err)
		}
	case <-rootCtx.Done():
		log.Info("signal received")
	}

	return shutdown(srv, hub, mgr, log)
}

// shutdown drives the three-stage termination sequence. Each stage
// gets its own bounded ctx so a misbehaving step cannot eat the next
// one's budget.
func shutdown(srv httpx.Server, hub ws.Hub, mgr session.SessionManager, log *slog.Logger) error {
	httpCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(httpCtx); err != nil {
		log.Warn("http shutdown", "err", err)
	}

	if err := hub.Close(); err != nil {
		log.Warn("hub close", "err", err)
	}

	mgrCtx, cancelMgr := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelMgr()
	if err := mgr.Close(mgrCtx); err != nil {
		log.Warn("session close", "err", err)
	}

	log.Info("goodbye")
	return nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

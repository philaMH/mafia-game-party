// Package httpx is U4's HTTP server library. It exposes a minimal Server
// abstraction over net/http with the routes required by the LAN-only
// mafia-game PoC: a SPA fallback for the React app, an immutable-cache
// asset handler, the WebSocket upgrade endpoint (delegated to ws.Hub),
// /api/results, and /healthz.
//
// The package depends only on the standard library (NFR-U4-M4); the
// gorilla/websocket dependency is contained in U3 (internal/transport/ws).
//
// Conventions:
//   - All routes use net/http.ServeMux pattern matching (Go 1.22+).
//   - Logging middleware records 4 fields (method, path, status,
//     duration_ms) — payload bodies and query values are deliberately not
//     logged (NFR-U4-S2).
//   - /api/results omits Member.Token from the response (NFR-U4-S1).
//   - /assets/* uses immutable cache headers; /index.html is no-cache.
//   - Composition is driven by cmd/mafia-game/main.go; this package
//     never spawns goroutines or holds long-lived state besides the
//     embedded http.Server.
package httpx

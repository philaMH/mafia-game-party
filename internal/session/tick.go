package session

import (
	"context"
	"log/slog"
	"time"
)

// tickLoop fires Tick on s.opts.TickInterval cadence until stopCh closes.
// It runs in its own goroutine; see BR-U2-TICK-*.
func (s *session) tickLoop() {
	defer s.tickerWG.Done()

	ticker := time.NewTicker(s.opts.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case now := <-ticker.C:
			s.Tick(now)
		}
	}
}

// Tick advances time-driven phases. It is a no-op when no game is active.
// Caller MUST NOT hold s.mu (the loop acquires it internally).
func (s *session) Tick(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.sess.Started {
		return
	}
	state, envs, err := s.engine.Tick(now)
	if err != nil {
		slog.Error("engine.Tick failed", "err", err)
		return
	}
	if len(envs) == 0 {
		return
	}
	s.persistAndDispatch(context.Background(), state, envs)
}

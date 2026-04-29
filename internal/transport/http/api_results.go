package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/saltware/mafia-game/internal/game"
	"github.com/saltware/mafia-game/internal/persistence"
)

// resultsResponse is the top-level shape of GET /api/results.
type resultsResponse struct {
	Results []resultEntry `json:"results"`
}

// resultEntry mirrors persistence.GameResult on the wire — minus
// Members[*].Token, which never crosses the API boundary (NFR-U4-S1).
type resultEntry struct {
	GameID    string        `json:"gameId"`
	StartedAt time.Time     `json:"startedAt"`
	EndedAt   time.Time     `json:"endedAt"`
	Winner    *game.Team    `json:"winner"`
	EndReason string        `json:"endReason"`
	Options   game.Options  `json:"options"`
	Members   []memberEntry `json:"members"`
	Reveal    []game.Player `json:"reveal"`
}

// memberEntry is a deliberately Token-free projection of
// persistence.PersistedMember.
type memberEntry struct {
	ID       game.PlayerID `json:"id"`
	Name     string        `json:"name"`
	JoinedAt time.Time     `json:"joinedAt"`
}

const (
	defaultLimit = 50
	maxLimit     = 500
)

// resultsHandler returns the recent finished games as JSON. The query
// parameter `limit` is bounded to [1, 500]; out-of-range values are
// rejected with 400 to keep clients honest.
func resultsHandler(store persistence.PersistenceStore, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := defaultLimit
		if v := r.URL.Query().Get("limit"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 || n > maxLimit {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
			limit = n
		}

		results, err := store.ListResults(r.Context(), limit)
		if err != nil {
			log.Error("ListResults", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		resp := buildResultsResponse(results)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Error("encode results", "err", err)
		}
	}
}

// buildResultsResponse converts persistence.GameResult slices into the
// wire-friendly resultsResponse, dropping Member.Token entirely.
func buildResultsResponse(results []persistence.GameResult) resultsResponse {
	resp := resultsResponse{Results: make([]resultEntry, 0, len(results))}
	for _, r := range results {
		members := make([]memberEntry, 0, len(r.Members))
		for _, m := range r.Members {
			members = append(members, memberEntry{
				ID:       m.ID,
				Name:     m.Name,
				JoinedAt: m.JoinedAt,
			})
		}
		resp.Results = append(resp.Results, resultEntry{
			GameID:    r.GameID,
			StartedAt: r.StartedAt,
			EndedAt:   r.EndedAt,
			Winner:    r.Winner,
			EndReason: string(r.EndReason),
			Options:   r.Options,
			Members:   members,
			Reveal:    r.Reveal,
		})
	}
	return resp
}

package announce

import "github.com/saltware/mafia-game/internal/game"

// Severity classifies announcements for the public-screen UI. Severity is
// not a semantic guarantee — it is a styling hint (color, sound, weight).
type Severity string

// Severity values.
const (
	SeverityInfo     Severity = "INFO"
	SeverityEmphasis Severity = "EMPHASIS"
	SeverityWarn     Severity = "WARN"
)

// Announcement is the rendered output for a single event. Subtitle and
// Speech are intentionally identical at v1; future tweaks may diverge.
//
// An empty Announcement (Subtitle == "") signals "no message"; this is the
// expected output for private events such as RoleRevealedToPlayer or
// MafiaCohortRevealed (BR-U2-CAT-1).
type Announcement struct {
	Subtitle      string
	Speech        string
	Severity      Severity
	ForPublicOnly bool
}

// IsEmpty reports whether the announcement should not be displayed.
func (a Announcement) IsEmpty() bool { return a.Subtitle == "" }

// CatalogContext supplies session-scoped lookups required for variable
// interpolation. The session manager constructs one per Render call so
// member name changes and current settings between events are reflected.
type CatalogContext struct {
	GetName               func(id game.PlayerID) string
	IntroSecondsPerPlayer int
}

// nameOf is a convenience helper that falls back to the raw PlayerID when
// no lookup function is provided.
func (c CatalogContext) nameOf(id game.PlayerID) string {
	if c.GetName != nil {
		return c.GetName(id)
	}
	return string(id)
}

// AnnouncementCatalog is the FR-7.2 abstraction: render an envelope (or an
// engine error) into a Korean Announcement. Implementations must be safe to
// call concurrently from multiple SessionManager instances; the bundled
// defaultCatalog is stateless.
type AnnouncementCatalog interface {
	Render(env game.EventEnvelope, ctx CatalogContext) Announcement
	RenderError(err error, sender game.PlayerID, ctx CatalogContext) Announcement
}

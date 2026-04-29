// Package announce maps domain events from internal/game into Korean
// announcement strings used as both subtitle and TTS speech (FR-7.2,
// FR-8.3, FR-8.4). The catalog is exposed behind the AnnouncementCatalog
// interface so callers can swap implementations (e.g., for tests or future
// localization). All public-facing text is intentionally formal/grave to
// match the FD's "근엄 톤" decision (Q-FD-U2-8=A).
package announce

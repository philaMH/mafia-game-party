package session

import (
	"context"

	"github.com/saltware/mafia-game/internal/game"
)

// SaveHostOptions implements SessionManager. The call is host-token gated
// (single-host invariant) and shape-validated. Successful calls overwrite
// any previously saved options.
func (s *session) SaveHostOptions(ctx context.Context, token HostToken, opts game.Options) error {
	_ = ctx
	if err := s.hostAuth.Verify(token); err != nil {
		return err
	}
	if err := validateSavedHostOptions(opts); err != nil {
		return err
	}

	s.mu.Lock()
	s.savedHostOptions = opts
	s.hasSavedHostOptions = true
	s.mu.Unlock()
	return nil
}

// SavedHostOptions implements SessionManager. Returns a copy so the caller
// cannot mutate internal state.
func (s *session) SavedHostOptions() (game.Options, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.savedHostOptions, s.hasSavedHostOptions
}

// validateSavedHostOptions performs a shape-only check of the host
// options entered via the settings screen. It accumulates every
// violation so the UI can surface them all at once, mirroring the
// game.validateOptions pattern. Player-count-dependent invariants are
// deliberately deferred to Engine.Start.
func validateSavedHostOptions(opts game.Options) error {
	var errs game.ValidationErrors

	if opts.MaxPlayers < 6 || opts.MaxPlayers > 12 {
		errs = append(errs, game.FieldError{
			Field:   "maxPlayers",
			Code:    game.CodeValidation,
			Message: "must be in [6,12]",
		})
	}
	if opts.MafiaCount < 1 {
		errs = append(errs, game.FieldError{
			Field:   "mafiaCount",
			Code:    game.CodeValidation,
			Message: "must be >= 1",
		})
	}
	// Citizen-side guard: doctor + police + at least 1 plain citizen.
	if opts.MaxPlayers >= 6 && opts.MafiaCount >= 1 && opts.MafiaCount > opts.MaxPlayers-3 {
		errs = append(errs, game.FieldError{
			Field:   "mafiaCount",
			Code:    game.CodeValidation,
			Message: "mafiaCount must be <= maxPlayers - 3",
		})
	}
	if opts.IntroSecondsPerPlayer < 5 {
		errs = append(errs, game.FieldError{
			Field:   "introSecondsPerPlayer",
			Code:    game.CodeValidation,
			Message: "must be >= 5",
		})
	}
	if opts.DiscussionSeconds < 30 {
		errs = append(errs, game.FieldError{
			Field:   "discussionSeconds",
			Code:    game.CodeValidation,
			Message: "must be >= 30",
		})
	}
	if opts.NightMafiaSeconds < 5 {
		errs = append(errs, game.FieldError{
			Field:   "nightMafiaSeconds",
			Code:    game.CodeValidation,
			Message: "must be >= 5",
		})
	}
	if opts.NightPoliceSeconds < 5 {
		errs = append(errs, game.FieldError{
			Field:   "nightPoliceSeconds",
			Code:    game.CodeValidation,
			Message: "must be >= 5",
		})
	}
	if opts.NightDoctorSeconds < 5 {
		errs = append(errs, game.FieldError{
			Field:   "nightDoctorSeconds",
			Code:    game.CodeValidation,
			Message: "must be >= 5",
		})
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

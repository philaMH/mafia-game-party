package game

import (
	"fmt"
	"strings"
)

// FieldError is a single accumulated field-level validation failure.
type FieldError struct {
	Field   string
	Code    ErrorCode
	Message string
}

// Error implements error for a single field-level failure.
func (e FieldError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field=%s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// ValidationErrors is the typed error returned by validators that accumulate
// multiple violations (e.g. validateOptions). Callers may surface every
// violation at once instead of fail-fast.
type ValidationErrors []FieldError

// Error joins all field errors with semicolons.
func (e ValidationErrors) Error() string {
	parts := make([]string, len(e))
	for i, fe := range e {
		parts[i] = fe.Error()
	}
	return strings.Join(parts, "; ")
}

// Is reports a match if target is *EngineError with CodeValidation, allowing
// callers to write errors.Is(err, ErrValidation) regardless of single vs
// accumulated form.
func (e ValidationErrors) Is(target error) bool {
	t, ok := target.(*EngineError)
	if !ok {
		return false
	}
	return t.Code == "" || t.Code == CodeValidation
}

// validateOptions checks the host-supplied options against BR-OPT-1..8.
// Returns nil when all rules pass; otherwise returns a ValidationErrors with
// every violation accumulated so the host UI can show them all at once.
func validateOptions(opts Options, playerCount int) error {
	var errs ValidationErrors

	if playerCount < 6 || playerCount > 12 {
		errs = append(errs, FieldError{Field: "playerCount", Code: CodeValidation,
			Message: fmt.Sprintf("must be in [6,12]; got %d", playerCount)})
	}
	if opts.MaxPlayers != 0 {
		if opts.MaxPlayers < 6 || opts.MaxPlayers > 12 {
			errs = append(errs, FieldError{Field: "maxPlayers", Code: CodeValidation,
				Message: fmt.Sprintf("must be in [6,12]; got %d", opts.MaxPlayers)})
		} else if playerCount > opts.MaxPlayers {
			errs = append(errs, FieldError{Field: "maxPlayers", Code: CodeValidation,
				Message: fmt.Sprintf("actual players %d exceeds maxPlayers %d", playerCount, opts.MaxPlayers)})
		}
	}
	if opts.MafiaCount < 1 {
		errs = append(errs, FieldError{Field: "mafiaCount", Code: CodeValidation,
			Message: fmt.Sprintf("must be >= 1; got %d", opts.MafiaCount)})
	}
	citizenSide := playerCount - opts.MafiaCount
	if opts.MafiaCount >= 1 && citizenSide < opts.MafiaCount+1 {
		errs = append(errs, FieldError{Field: "mafiaCount", Code: CodeValidation,
			Message: fmt.Sprintf("citizen-side must exceed mafia by >= 1 (need >= %d, got %d)",
				opts.MafiaCount+1, citizenSide)})
	}
	// Doctor + Police are fixed at 1 each; ensure at least 1 plain CITIZEN remains.
	if opts.MafiaCount >= 1 && (citizenSide-2) < 1 {
		errs = append(errs, FieldError{Field: "mafiaCount", Code: CodeValidation,
			Message: fmt.Sprintf("at least 1 plain citizen required (got %d)", citizenSide-2)})
	}
	if opts.IntroSecondsPerPlayer < 5 {
		errs = append(errs, FieldError{Field: "introSecondsPerPlayer", Code: CodeValidation,
			Message: fmt.Sprintf("must be >= 5; got %d", opts.IntroSecondsPerPlayer)})
	}
	if opts.DiscussionSeconds < 30 {
		errs = append(errs, FieldError{Field: "discussionSeconds", Code: CodeValidation,
			Message: fmt.Sprintf("must be >= 30; got %d", opts.DiscussionSeconds)})
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// ensureHost returns ErrPermissionDenied if sender is not the host.
func ensureHost(s *State, sender PlayerID) error {
	if sender != s.HostID {
		return errf(CodePermissionDenied, "sender %q is not host %q", sender, s.HostID)
	}
	return nil
}

// ensurePhase returns ErrWrongPhase if the current phase is not in allowed.
func ensurePhase(s *State, allowed ...Phase) error {
	for _, p := range allowed {
		if s.Phase == p {
			return nil
		}
	}
	return errf(CodeWrongPhase, "phase=%s not in %v", s.Phase, allowed)
}

// ensureRole returns ErrRoleMismatch if the sender does not hold the role.
func ensureRole(s *State, sender PlayerID, role Role) error {
	p, ok := s.FindPlayer(sender)
	if !ok {
		return errf(CodeUnknownPlayer, "unknown player %q", sender)
	}
	if p.Role != role {
		return errf(CodeRoleMismatch, "player %q is not %s", sender, role)
	}
	return nil
}

// ensureAlive returns ErrDeadPlayer if any of the IDs is not alive (or
// unknown).
func ensureAlive(s *State, ids ...PlayerID) error {
	for _, id := range ids {
		p, ok := s.FindPlayer(id)
		if !ok {
			return errf(CodeUnknownPlayer, "unknown player %q", id)
		}
		if !p.Alive {
			return errf(CodeDeadPlayer, "player %q is not alive", id)
		}
	}
	return nil
}

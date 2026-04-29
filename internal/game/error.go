package game

import "fmt"

// ErrorCode classifies engine errors. Callers compare codes via errors.Is
// against the package-level sentinel errors below.
type ErrorCode string

// Error code constants.
const (
	CodeValidation        ErrorCode = "VALIDATION_ERROR"
	CodeWrongPhase        ErrorCode = "WRONG_PHASE_ERROR"
	CodePermissionDenied  ErrorCode = "PERMISSION_DENIED_ERROR"
	CodeRoleMismatch      ErrorCode = "ROLE_MISMATCH_ERROR"
	CodeNotRepresentative ErrorCode = "NOT_REPRESENTATIVE_ERROR"
	CodeDeadPlayer        ErrorCode = "DEAD_PLAYER_ERROR"
	CodeAlreadyDone       ErrorCode = "ALREADY_DONE_ERROR"
	CodeInvalidTarget     ErrorCode = "INVALID_TARGET_ERROR"
	CodeUnknownPlayer     ErrorCode = "UNKNOWN_PLAYER_ERROR"
)

// EngineError is the typed error returned by Engine.Apply for single-rule
// violations. It supports errors.Is / errors.As by code.
type EngineError struct {
	Code    ErrorCode
	Message string
	// Field is optional context: the offending field name when available.
	Field string
}

// Error implements the error interface. The message is for developers; user-
// facing announcements are produced by the AnnouncementService unit.
func (e *EngineError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: %s (field=%s)", e.Code, e.Message, e.Field)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Is matches by code only, ignoring message and field. This lets callers
// write errors.Is(err, ErrValidation) without caring about the specifics.
func (e *EngineError) Is(target error) bool {
	t, ok := target.(*EngineError)
	if !ok {
		return false
	}
	if t.Code == "" {
		// Sentinel without a code matches any EngineError.
		return true
	}
	return e.Code == t.Code
}

// Sentinel errors. Use with errors.Is to identify error categories.
var (
	ErrValidation        = &EngineError{Code: CodeValidation}
	ErrWrongPhase        = &EngineError{Code: CodeWrongPhase}
	ErrPermissionDenied  = &EngineError{Code: CodePermissionDenied}
	ErrRoleMismatch      = &EngineError{Code: CodeRoleMismatch}
	ErrNotRepresentative = &EngineError{Code: CodeNotRepresentative}
	ErrDeadPlayer        = &EngineError{Code: CodeDeadPlayer}
	ErrAlreadyDone       = &EngineError{Code: CodeAlreadyDone}
	ErrInvalidTarget     = &EngineError{Code: CodeInvalidTarget}
	ErrUnknownPlayer     = &EngineError{Code: CodeUnknownPlayer}
)

// errf constructs a fresh EngineError instance with the given code and message.
// Use this for error returns; do not return the sentinel pointers directly
// (they are meant for matching only).
func errf(code ErrorCode, format string, args ...any) *EngineError {
	return &EngineError{Code: code, Message: fmt.Sprintf(format, args...)}
}

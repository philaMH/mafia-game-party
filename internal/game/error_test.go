package game

import (
	"errors"
	"testing"
)

func TestEngineError_Is(t *testing.T) {
	err := errf(CodeValidation, "invalid")
	if !errors.Is(err, ErrValidation) {
		t.Errorf("errors.Is(err, ErrValidation) should be true")
	}
	if errors.Is(err, ErrPermissionDenied) {
		t.Errorf("errors.Is(err, ErrPermissionDenied) should be false for validation")
	}
}

func TestEngineError_As(t *testing.T) {
	var err error = errf(CodeRoleMismatch, "wrong role")
	var ee *EngineError
	if !errors.As(err, &ee) {
		t.Fatalf("errors.As should succeed")
	}
	if ee.Code != CodeRoleMismatch {
		t.Errorf("code=%s, want %s", ee.Code, CodeRoleMismatch)
	}
}

func TestValidationErrors_Format(t *testing.T) {
	ve := ValidationErrors{
		{Field: "a", Code: CodeValidation, Message: "bad a"},
		{Field: "b", Code: CodeValidation, Message: "bad b"},
	}
	got := ve.Error()
	if got == "" {
		t.Errorf("ValidationErrors.Error empty")
	}
	if !errors.Is(ve, ErrValidation) {
		t.Errorf("ValidationErrors.Is should match ErrValidation")
	}
}

func TestEngineError_NilSafe(t *testing.T) {
	var e *EngineError
	if got := e.Error(); got != "<nil>" {
		t.Errorf("nil EngineError.Error()=%q, want <nil>", got)
	}
}

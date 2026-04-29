package announce

import (
	"errors"
	"fmt"
	"strings"

	"github.com/saltware/mafia-game/internal/game"
)

// Korean user-facing error messages keyed by EngineError code.
// (BR-U2-ERR-1, domain-entities.md §5 ErrorAnnouncement table.)
var errorMessages = map[game.ErrorCode]string{
	game.CodeValidation:        "입력이 올바르지 않습니다",
	game.CodeWrongPhase:        "지금은 그 행동을 할 수 없습니다.",
	game.CodePermissionDenied:  "권한이 없습니다.",
	game.CodeRoleMismatch:      "당신의 역할은 그 행동을 할 수 없습니다.",
	game.CodeNotRepresentative: "이번 게임의 마피아 대표자만 살해 대상을 입력할 수 있습니다.",
	game.CodeDeadPlayer:        "사망한 플레이어는 행동할 수 없습니다.",
	game.CodeAlreadyDone:       "이번 단계에서는 이미 행동을 완료했습니다.",
	game.CodeInvalidTarget:     "선택할 수 없는 대상입니다.",
	game.CodeUnknownPlayer:     "알 수 없는 플레이어입니다.",
}

const errFallback = "알 수 없는 오류가 발생했습니다."

// RenderError implements AnnouncementCatalog. The result is intended for the
// sender's PlayerView only (BR-U2-ERR-6); ForPublicOnly is therefore false.
func (defaultCatalog) RenderError(err error, sender game.PlayerID, ctx CatalogContext) Announcement {
	_ = sender // routing is performed by the SessionManager; reserved here.
	_ = ctx
	if err == nil {
		return Announcement{}
	}

	// errors.As handles wrapped errors and ValidationErrors aggregates.
	var ve game.ValidationErrors
	if errors.As(err, &ve) {
		return renderValidationErrors(ve)
	}

	var ee *game.EngineError
	if errors.As(err, &ee) {
		base, ok := errorMessages[ee.Code]
		if !ok {
			base = errFallback
		}
		// Validation errors include the offending field name when available.
		if ee.Code == game.CodeValidation && ee.Field != "" {
			base = fmt.Sprintf("%s: %s", base, ee.Field)
		} else if ee.Code == game.CodeValidation {
			base = base + "."
		}
		return Announcement{
			Subtitle:      base,
			Speech:        base,
			Severity:      severityForCode(ee.Code),
			ForPublicOnly: false,
		}
	}

	return Announcement{
		Subtitle:      errFallback,
		Speech:        errFallback,
		Severity:      SeverityWarn,
		ForPublicOnly: false,
	}
}

// renderValidationErrors joins multiple field errors with "; ".
func renderValidationErrors(ve game.ValidationErrors) Announcement {
	parts := make([]string, 0, len(ve))
	for _, fe := range ve {
		parts = append(parts, fmt.Sprintf("%s: %s", fe.Field, fe.Message))
	}
	combined := "입력이 올바르지 않습니다 — " + strings.Join(parts, "; ")
	return Announcement{
		Subtitle:      combined,
		Speech:        combined,
		Severity:      SeverityWarn,
		ForPublicOnly: false,
	}
}

// severityForCode picks a UI severity per error category.
func severityForCode(code game.ErrorCode) Severity {
	switch code {
	case game.CodeValidation, game.CodeInvalidTarget:
		return SeverityWarn
	case game.CodePermissionDenied, game.CodeRoleMismatch,
		game.CodeNotRepresentative, game.CodeDeadPlayer, game.CodeAlreadyDone:
		return SeverityWarn
	default:
		return SeverityWarn
	}
}

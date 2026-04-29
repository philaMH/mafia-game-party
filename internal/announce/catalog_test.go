package announce_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/saltware/mafia-game/internal/announce"
	"github.com/saltware/mafia-game/internal/game"
)

func ctx() announce.CatalogContext {
	return announce.CatalogContext{
		GetName: func(id game.PlayerID) string {
			switch id {
			case "p1":
				return "철수"
			case "p2":
				return "영희"
			}
			return string(id)
		},
		IntroSecondsPerPlayer: 30,
	}
}

func render(t *testing.T, e game.Event, vis game.Visibility) announce.Announcement {
	t.Helper()
	return announce.NewDefaultCatalog().Render(
		game.EventEnvelope{Event: e, Visibility: vis},
		ctx(),
	)
}

func TestRender_GameStarted(t *testing.T) {
	a := render(t, game.GameStarted{}, game.VisPublic)
	if a.IsEmpty() {
		t.Fatal("expected non-empty")
	}
	if a.Severity != announce.SeverityEmphasis {
		t.Errorf("severity = %v", a.Severity)
	}
	if !strings.Contains(a.Subtitle, "마피아 게임") {
		t.Errorf("subtitle wrong: %q", a.Subtitle)
	}
	if a.Subtitle != a.Speech {
		t.Errorf("subtitle != speech")
	}
	if !a.ForPublicOnly {
		t.Error("public events must be ForPublicOnly")
	}
}

func TestRender_PhaseChangedAllPhases(t *testing.T) {
	cases := []struct {
		phase   game.Phase
		want    string
		sev     announce.Severity
		nonEmpt bool
	}{
		{game.PhaseIntro, "30초", announce.SeverityInfo, true},
		{game.PhaseNight, "밤이", announce.SeverityEmphasis, true},
		{game.PhaseDay, "아침", announce.SeverityEmphasis, true},
		{game.PhaseVote, "표를", announce.SeverityEmphasis, true},
		{game.PhaseRecount, "마지막", announce.SeverityWarn, true},
		{game.PhaseLobby, "", announce.Severity(""), false},
		{game.PhaseEnd, "", announce.Severity(""), false},
	}
	for _, tc := range cases {
		t.Run(string(tc.phase), func(t *testing.T) {
			a := render(t, game.PhaseChanged{Phase: tc.phase, Day: 2}, game.VisPublic)
			if tc.nonEmpt {
				if a.IsEmpty() {
					t.Fatal("expected non-empty")
				}
				if !strings.Contains(a.Subtitle, tc.want) {
					t.Errorf("subtitle %q missing %q", a.Subtitle, tc.want)
				}
				if a.Severity != tc.sev {
					t.Errorf("severity got %v want %v", a.Severity, tc.sev)
				}
			} else if !a.IsEmpty() {
				t.Errorf("expected empty for phase=%s, got %+v", tc.phase, a)
			}
		})
	}
}

func TestRender_IntroSpeakerInterpolatesName(t *testing.T) {
	a := render(t, game.IntroSpeakerChanged{PlayerID: "p1"}, game.VisPublic)
	if !strings.Contains(a.Subtitle, "철수") {
		t.Errorf("expected 철수 in %q", a.Subtitle)
	}
}

func TestRender_DeathInterpolates(t *testing.T) {
	a := render(t, game.DeathAnnounced{Victim: "p2"}, game.VisPublic)
	if !strings.Contains(a.Subtitle, "영희") {
		t.Errorf("expected 영희 in %q", a.Subtitle)
	}
}

func TestRender_PeacefulNight(t *testing.T) {
	a := render(t, game.PeacefulNight{}, game.VisPublic)
	if !strings.Contains(a.Subtitle, "사망") {
		t.Errorf("subtitle wrong: %q", a.Subtitle)
	}
}

func TestRender_Day1UsesDedicatedSubtitle(t *testing.T) {
	a := render(t, game.PhaseChanged{Phase: game.PhaseDay, Day: 1}, game.VisPublic)
	if a.IsEmpty() {
		t.Fatal("Day 1 PhaseChanged must have a subtitle")
	}
	if !strings.Contains(a.Subtitle, "첫째 날") {
		t.Errorf("Day 1 subtitle should reference 첫째 날, got %q", a.Subtitle)
	}
	a2 := render(t, game.PhaseChanged{Phase: game.PhaseDay, Day: 2}, game.VisPublic)
	if !strings.Contains(a2.Subtitle, "2일째") {
		t.Errorf("Day 2 subtitle should reference 2일째, got %q", a2.Subtitle)
	}
}

func TestRender_NightStepChanged(t *testing.T) {
	cases := []struct {
		step game.NightStep
		want string
	}{
		{game.NightStepMafia, "마피아"},
		{game.NightStepPolice, "경찰"},
		{game.NightStepDoctor, "의사"},
	}
	for _, tc := range cases {
		t.Run(string(tc.step), func(t *testing.T) {
			a := render(t, game.NightStepChanged{Step: tc.step, Day: 1}, game.VisPublic)
			if a.IsEmpty() {
				t.Fatalf("step %s should produce a subtitle", tc.step)
			}
			if !strings.Contains(a.Subtitle, tc.want) {
				t.Errorf("subtitle %q missing %q", a.Subtitle, tc.want)
			}
		})
	}
	a := render(t, game.NightStepChanged{Step: game.NightStepResolved}, game.VisPublic)
	if !a.IsEmpty() {
		t.Errorf("RESOLVED step should be silent, got %+v", a)
	}
}

func TestRender_GamePausedAndResumed(t *testing.T) {
	paused := render(t, game.GamePaused{Phase: game.PhaseNight}, game.VisPublic)
	if paused.IsEmpty() {
		t.Fatalf("GamePaused should render a subtitle")
	}
	if !strings.Contains(paused.Subtitle, "멈") {
		t.Errorf("GamePaused subtitle missing 멈... fragment: %q", paused.Subtitle)
	}
	resumed := render(t, game.GameResumed{Phase: game.PhaseNight}, game.VisPublic)
	if resumed.IsEmpty() {
		t.Fatalf("GameResumed should render a subtitle")
	}
	if !strings.Contains(resumed.Subtitle, "이어") && !strings.Contains(resumed.Subtitle, "다시") {
		t.Errorf("GameResumed subtitle missing 이어/다시: %q", resumed.Subtitle)
	}
}

func TestRender_EliminatedIncludesRoleKr(t *testing.T) {
	cases := []struct {
		role game.Role
		want string
	}{
		{game.RoleMafia, "마피아"},
		{game.RoleDoctor, "의사"},
		{game.RolePolice, "경찰"},
		{game.RoleCitizen, "시민"},
	}
	for _, tc := range cases {
		t.Run(string(tc.role), func(t *testing.T) {
			a := render(t, game.Eliminated{PlayerID: "p1", Role: tc.role}, game.VisPublic)
			if !strings.Contains(a.Subtitle, tc.want) {
				t.Errorf("missing %q in %q", tc.want, a.Subtitle)
			}
		})
	}
}

func TestRender_DiscussionTimerThresholds(t *testing.T) {
	for _, sl := range []int{30, 10, 0} {
		a := render(t, game.DiscussionTimerTick{SecondsLeft: sl}, game.VisPublic)
		if a.IsEmpty() {
			t.Errorf("expected non-empty for SecondsLeft=%d", sl)
		}
	}
	a := render(t, game.DiscussionTimerTick{SecondsLeft: 25}, game.VisPublic)
	if !a.IsEmpty() {
		t.Error("non-threshold tick should be empty")
	}
}

func TestRender_VoteTallied(t *testing.T) {
	target := game.PlayerID("p2")
	t.Run("recount", func(t *testing.T) {
		a := render(t, game.VoteTallied{Recount: true}, game.VisPublic)
		if !strings.Contains(a.Subtitle, "재투표") {
			t.Errorf("subtitle wrong: %q", a.Subtitle)
		}
	})
	t.Run("noElim", func(t *testing.T) {
		a := render(t, game.VoteTallied{}, game.VisPublic)
		if !strings.Contains(a.Subtitle, "처형이 없습니다") {
			t.Errorf("subtitle wrong: %q", a.Subtitle)
		}
	})
	t.Run("withElim_silent", func(t *testing.T) {
		a := render(t, game.VoteTallied{Eliminated: &target}, game.VisPublic)
		if !a.IsEmpty() {
			t.Errorf("expected empty (Eliminated event carries the speech), got %+v", a)
		}
	})
}

func TestRender_GameEnded(t *testing.T) {
	winner := game.TeamMafia
	cases := []struct {
		reason game.EndReason
		need   string
	}{
		{game.EndMafiaWin, "마피아의 승리"},
		{game.EndCitizenWin, "시민의 승리"},
		{game.EndHostForceEnd, "진행자의 결정"},
	}
	for _, tc := range cases {
		t.Run(string(tc.reason), func(t *testing.T) {
			a := render(t, game.GameEnded{Winner: &winner, EndReason: tc.reason}, game.VisPublic)
			if !strings.Contains(a.Subtitle, tc.need) {
				t.Errorf("subtitle %q missing %q", a.Subtitle, tc.need)
			}
		})
	}
}

func TestRender_VoiceToggled(t *testing.T) {
	on := render(t, game.VoiceToggled{On: true}, game.VisPublic)
	if !strings.Contains(on.Subtitle, "활성화") {
		t.Errorf("on: %q", on.Subtitle)
	}
	off := render(t, game.VoiceToggled{On: false}, game.VisPublic)
	if !strings.Contains(off.Subtitle, "비활성화") {
		t.Errorf("off: %q", off.Subtitle)
	}
}

func TestRender_PrivateEventsAreEmpty(t *testing.T) {
	cases := []game.Event{
		game.RoleRevealedToPlayer{PlayerID: "p1", Role: game.RoleMafia, Keyword: "kw"},
		game.MafiaCohortRevealed{MafiaIDs: []game.PlayerID{"p1"}, RepresentativeID: "p1"},
		game.MafiaTargetSelected{RepresentativeID: "p1", Target: "p2"},
		game.PoliceResult{Police: "p1", Target: "p2", Team: game.TeamMafia},
		game.MafiaRepresentativeReassigned{OldID: "p1", NewID: "p2"},
	}
	for _, e := range cases {
		a := announce.NewDefaultCatalog().Render(
			game.EventEnvelope{Event: e, Visibility: game.VisRoleMafia, PlayerID: "p1"},
			ctx(),
		)
		if !a.IsEmpty() {
			t.Errorf("private event %T must be empty, got %+v", e, a)
		}
	}
}

func TestRenderError_AllNineCodes(t *testing.T) {
	cases := []struct {
		code game.ErrorCode
		want string
	}{
		{game.CodeValidation, "올바르지"},
		{game.CodeWrongPhase, "할 수 없습니다"},
		{game.CodePermissionDenied, "권한이"},
		{game.CodeRoleMismatch, "역할은"},
		{game.CodeNotRepresentative, "마피아 대표자"},
		{game.CodeDeadPlayer, "사망한"},
		{game.CodeAlreadyDone, "이미 행동"},
		{game.CodeInvalidTarget, "선택할 수 없"},
		{game.CodeUnknownPlayer, "알 수 없는 플레이어"},
	}
	for _, tc := range cases {
		t.Run(string(tc.code), func(t *testing.T) {
			err := &game.EngineError{Code: tc.code, Message: "x"}
			a := announce.NewDefaultCatalog().RenderError(err, "p1", ctx())
			if a.IsEmpty() {
				t.Fatalf("expected non-empty for code %s", tc.code)
			}
			if !strings.Contains(a.Subtitle, tc.want) {
				t.Errorf("expected %q in %q", tc.want, a.Subtitle)
			}
			if a.ForPublicOnly {
				t.Error("error annoucements should be private (ForPublicOnly=false)")
			}
		})
	}
}

func TestRenderError_ValidationFieldInterpolated(t *testing.T) {
	err := &game.EngineError{Code: game.CodeValidation, Field: "name", Message: "duplicate"}
	a := announce.NewDefaultCatalog().RenderError(err, "p1", ctx())
	if !strings.Contains(a.Subtitle, "name") {
		t.Errorf("expected field name in %q", a.Subtitle)
	}
}

func TestRenderError_ValidationErrorsAggregate(t *testing.T) {
	ve := game.ValidationErrors{
		{Field: "f1", Code: game.CodeValidation, Message: "bad"},
		{Field: "f2", Code: game.CodeValidation, Message: "worse"},
	}
	a := announce.NewDefaultCatalog().RenderError(ve, "p1", ctx())
	if !strings.Contains(a.Subtitle, "f1") || !strings.Contains(a.Subtitle, "f2") {
		t.Errorf("expected both fields in %q", a.Subtitle)
	}
}

func TestRenderError_NilReturnsEmpty(t *testing.T) {
	a := announce.NewDefaultCatalog().RenderError(nil, "", ctx())
	if !a.IsEmpty() {
		t.Errorf("nil error should render empty, got %+v", a)
	}
}

func TestRenderError_UnknownErrorFallback(t *testing.T) {
	a := announce.NewDefaultCatalog().RenderError(errors.New("boom"), "p1", ctx())
	if a.IsEmpty() {
		t.Error("unknown error should still render")
	}
}

func TestSystemHelpers(t *testing.T) {
	r := announce.SystemRestore()
	if r.IsEmpty() || !strings.Contains(r.Subtitle, "복원") {
		t.Errorf("SystemRestore wrong: %+v", r)
	}
	pf := announce.SystemPersistFailure()
	if pf.IsEmpty() || pf.Severity != announce.SeverityWarn {
		t.Errorf("SystemPersistFailure wrong: %+v", pf)
	}
}

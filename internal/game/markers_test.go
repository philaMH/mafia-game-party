package game

import "testing"

// TestActionInterfaceImplementations confirms every action type satisfies
// the Action sealed interface. The seal is enforced via embedded
// sealedAction; this assertion catches any type that drops the embed.
func TestActionInterfaceImplementations(t *testing.T) {
	var _ Action = StartGame{}
	var _ Action = AdvanceIntro{}
	var _ Action = SubmitMafiaKill{}
	var _ Action = SubmitDoctorHeal{}
	var _ Action = SubmitPoliceCheck{}
	var _ Action = EndNightEarly{}
	var _ Action = EndDiscussionEarly{}
	var _ Action = SubmitVote{}
	var _ Action = ToggleVoice{}
	var _ Action = ForceEndGame{}
	var _ Action = PauseGame{}
	var _ Action = ResumeGame{}
}

// TestEventInterfaceImplementations confirms every event type satisfies
// the Event sealed interface.
func TestEventInterfaceImplementations(t *testing.T) {
	var _ Event = PlayerJoined{}
	var _ Event = GameStarted{}
	var _ Event = PhaseChanged{}
	var _ Event = RoleRevealedToPlayer{}
	var _ Event = MafiaCohortRevealed{}
	var _ Event = IntroSpeakerChanged{}
	var _ Event = MafiaTargetSelected{}
	var _ Event = PoliceResult{}
	var _ Event = DeathAnnounced{}
	var _ Event = PeacefulNight{}
	var _ Event = DiscussionTimerTick{}
	var _ Event = VoteTallied{}
	var _ Event = Eliminated{}
	var _ Event = MafiaRepresentativeReassigned{}
	var _ Event = GameEnded{}
	var _ Event = VoiceToggled{}
	var _ Event = NightStepChanged{}
	var _ Event = GamePaused{}
	var _ Event = GameResumed{}
}

func TestPendingActions(t *testing.T) {
	pid := PlayerID("p1")
	s := State{PendingMafiaTarget: &pid, PendingDoctorTarget: &pid}
	pa := s.Pending()
	if pa.MafiaTarget == nil || *pa.MafiaTarget != "p1" {
		t.Errorf("Pending.MafiaTarget mismatch")
	}
	if pa.DoctorTarget == nil || *pa.DoctorTarget != "p1" {
		t.Errorf("Pending.DoctorTarget mismatch")
	}
}

func TestLivingMafiaIDs(t *testing.T) {
	s := State{Players: []Player{
		{ID: "a", Alive: true, Role: RoleMafia},
		{ID: "b", Alive: false, Role: RoleMafia},
		{ID: "c", Alive: true, Role: RoleCitizen},
	}}
	got := s.LivingMafiaIDs()
	if len(got) != 1 || got[0] != "a" {
		t.Errorf("LivingMafiaIDs=%v, want [a]", got)
	}
}

func TestNewDefaultBuildsEngine(t *testing.T) {
	e := NewDefault(NewDefaultKeywordPool())
	if e == nil {
		t.Fatalf("NewDefault returned nil")
	}
}

func TestRealClockReturnsCurrent(t *testing.T) {
	c := realClock{}
	if c.Now().IsZero() {
		t.Errorf("realClock.Now should be non-zero")
	}
}

func TestEngineString(t *testing.T) {
	e, _ := newTestEngine(t, 1)
	mustStart(t, e, playerSet(6), "p1", DefaultOptions(6))
	en := e.(*engine)
	if en.String() == "" {
		t.Errorf("engine.String empty")
	}
}

func TestEngineErrorWithField(t *testing.T) {
	e := &EngineError{Code: CodeValidation, Message: "bad", Field: "mafiaCount"}
	if got := e.Error(); got == "" {
		t.Errorf("Error() empty")
	}
}

func TestFieldErrorWithoutField(t *testing.T) {
	fe := FieldError{Code: CodeValidation, Message: "global"}
	if got := fe.Error(); got == "" {
		t.Errorf("Error() empty")
	}
}

func TestValidationErrorsIs_NotEngineError(t *testing.T) {
	ve := ValidationErrors{}
	type otherErr struct{ error }
	if ve.Is(otherErr{}) {
		t.Errorf("ValidationErrors.Is should not match unrelated error type")
	}
}

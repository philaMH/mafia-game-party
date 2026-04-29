package game

// Apply implements Engine. Dispatches to the per-action handler via type
// switch (P1). On any error, engine state is unchanged.
func (e *engine) Apply(action Action) (State, []EventEnvelope, error) {
	if e.state.Phase == PhaseEnd {
		return e.state.Clone(), nil, errf(CodeWrongPhase, "game has ended")
	}
	switch a := action.(type) {
	case StartGame:
		return e.handleStartGame(a)
	case AdvanceIntro:
		return e.handleAdvanceIntro(a)
	case EndSelfIntro:
		return e.handleEndSelfIntro(a)
	case SubmitMafiaKill:
		return e.handleMafiaKill(a)
	case SubmitDoctorHeal:
		return e.handleDoctorHeal(a)
	case SubmitPoliceCheck:
		return e.handlePoliceCheck(a)
	case EndNightEarly:
		return e.handleEndNightEarly(a)
	case EndDiscussionEarly:
		return e.handleEndDiscussionEarly(a)
	case SubmitVote:
		return e.handleVote(a)
	case ToggleVoice:
		return e.handleToggleVoice(a)
	case ForceEndGame:
		return e.handleForceEnd(a)
	case PauseGame:
		return e.handlePauseGame(a)
	case ResumeGame:
		return e.handleResumeGame(a)
	default:
		return e.state.Clone(), nil, errf(CodeValidation, "unknown action type %T", action)
	}
}


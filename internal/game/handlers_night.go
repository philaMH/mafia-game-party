package game

// handleMafiaKill is the mafia representative's nightly kill choice.
// Iteration 5 R3: the first submission within a NIGHT wins; further
// submissions are rejected with CodeAlreadyDone instead of last-write-
// wins. Submission no longer advances NightStep — only the timer in
// Tick does (Q1=A).
func (e *engine) handleMafiaKill(a SubmitMafiaKill) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseNight); err != nil {
		return e.state.Clone(), nil, err
	}
	if e.state.NightStep != NightStepMafia {
		return e.state.Clone(), nil, errf(CodeWrongPhase,
			"mafia kill rejected: night step is %q, not MAFIA", e.state.NightStep)
	}
	if err := ensureRole(&e.state, a.Mafia, RoleMafia); err != nil {
		return e.state.Clone(), nil, err
	}
	if err := ensureAlive(&e.state, a.Mafia, a.Target); err != nil {
		return e.state.Clone(), nil, err
	}
	if a.Mafia != e.state.MafiaRepresentativeID {
		return e.state.Clone(), nil, errf(CodeNotRepresentative,
			"player %q is not the mafia representative (%q)", a.Mafia, e.state.MafiaRepresentativeID)
	}
	if e.state.PendingMafiaTarget != nil {
		return e.state.Clone(), nil, errf(CodeAlreadyDone,
			"mafia kill already submitted this night")
	}
	target, _ := e.state.FindPlayer(a.Target)
	if target.Role == RoleMafia {
		return e.state.Clone(), nil, errf(CodeInvalidTarget, "mafia cannot target another mafia")
	}

	tgt := a.Target
	e.state.PendingMafiaTarget = &tgt
	events := []EventEnvelope{mafia(MafiaTargetSelected{
		RepresentativeID: a.Mafia,
		Target:           a.Target,
	})}
	return e.state.Clone(), events, nil
}

// handleDoctorHeal records the doctor's protect target. Iteration 5 R3:
// first submission wins; further submissions rejected. Submission does
// NOT trigger resolveNight — the DOCTOR step's deadline does.
func (e *engine) handleDoctorHeal(a SubmitDoctorHeal) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseNight); err != nil {
		return e.state.Clone(), nil, err
	}
	if e.state.NightStep != NightStepDoctor {
		return e.state.Clone(), nil, errf(CodeWrongPhase,
			"doctor heal rejected: night step is %q, not DOCTOR", e.state.NightStep)
	}
	if err := ensureRole(&e.state, a.Doctor, RoleDoctor); err != nil {
		return e.state.Clone(), nil, err
	}
	if err := ensureAlive(&e.state, a.Doctor, a.Target); err != nil {
		return e.state.Clone(), nil, err
	}
	if a.Doctor == a.Target && !e.state.Settings.DoctorSelfHealAllowed {
		return e.state.Clone(), nil, errf(CodeInvalidTarget, "self-heal disabled by options")
	}
	if e.state.PendingDoctorTarget != nil {
		return e.state.Clone(), nil, errf(CodeAlreadyDone,
			"doctor heal already submitted this night")
	}

	tgt := a.Target
	e.state.PendingDoctorTarget = &tgt
	return e.state.Clone(), nil, nil
}

// handlePoliceCheck performs a one-shot per-night investigation. Result is
// emitted privately to the police (Q-FD-U1-6=A team-only) and appended to
// State.PoliceHistory so it survives across nights and reconnections.
// Iteration 5: submission no longer advances NightStep (Tick handles
// transitions). The PoliceCheckedThisNight flag still enforces the
// once-per-night invariant.
func (e *engine) handlePoliceCheck(a SubmitPoliceCheck) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseNight); err != nil {
		return e.state.Clone(), nil, err
	}
	if e.state.NightStep != NightStepPolice {
		return e.state.Clone(), nil, errf(CodeWrongPhase,
			"police check rejected: night step is %q, not POLICE", e.state.NightStep)
	}
	if err := ensureRole(&e.state, a.Police, RolePolice); err != nil {
		return e.state.Clone(), nil, err
	}
	if err := ensureAlive(&e.state, a.Police, a.Target); err != nil {
		return e.state.Clone(), nil, err
	}
	if a.Police == a.Target {
		return e.state.Clone(), nil, errf(CodeInvalidTarget, "police cannot investigate self")
	}
	if e.state.PoliceCheckedThisNight {
		return e.state.Clone(), nil, errf(CodeAlreadyDone, "police already investigated this night")
	}

	tgt := a.Target
	e.state.PendingPoliceTarget = &tgt
	e.state.PoliceCheckedThisNight = true

	target, _ := e.state.FindPlayer(a.Target)
	team := TeamOf(target.Role)
	e.state.PoliceHistory = append(e.state.PoliceHistory, PoliceCheckRecord{
		Day:    e.state.Day,
		Target: a.Target,
		Team:   team,
	})

	events := []EventEnvelope{priv(PoliceResult{
		Police: a.Police,
		Target: a.Target,
		Team:   team,
	}, a.Police)}
	return e.state.Clone(), events, nil
}

// handleEndNightEarly: host forces NIGHT to end immediately, applying
// whichever night actions were submitted. Missing inputs are treated as
// "no action" (BR-RESOLVE-1/2). Iteration 5: with timer-driven step
// advancement the action is still kept as a maintenance hatch (e.g.,
// recovery scripts / tests) but the U5 host UI no longer surfaces it
// (Q4=A — only Pause/Resume on the host screen).
func (e *engine) handleEndNightEarly(a EndNightEarly) (State, []EventEnvelope, error) {
	if err := ensurePhase(&e.state, PhaseNight); err != nil {
		return e.state.Clone(), nil, err
	}
	if err := ensureHost(&e.state, a.HostID); err != nil {
		return e.state.Clone(), nil, err
	}
	events, err := e.resolveNight()
	if err != nil {
		return e.state.Clone(), nil, err
	}
	return e.state.Clone(), events, nil
}

package announce

import (
	"fmt"

	"github.com/saltware/mafia-game/internal/game"
)

// Stable audio cue identifiers. Each non-empty AudioID corresponds to a
// pre-recorded /audio/<id>.mp3 file the host client downloads on demand
// (Iter7 voice-script.md §3 + FR-8.9). Empty cues mean "subtitle only".
const (
	cueGameStarted     = "game.started"
	cuePhaseIntro      = "phase.intro"
	cuePhaseNight      = "phase.night"
	cuePhaseDayFirst   = "phase.day.first"
	cuePhaseDay        = "phase.day"
	cuePhaseVote       = "phase.vote"
	cuePhaseRecount    = "phase.recount"
	cueNightMafia      = "night.mafia"
	cueNightPolice     = "night.police"
	cueNightDoctor     = "night.doctor"
	cueIntroSpeaker    = "intro.speaker"
	cueDeathAnnounced  = "death.announced"
	cuePeacefulNight   = "peaceful.night"
	cueEliminatedMafia    = "eliminated.mafia"
	cueEliminatedNotMafia = "eliminated.notmafia"
	cueTimer30         = "timer.30"
	cueTimer10         = "timer.10"
	cueTimer0          = "timer.0"
	cueVoteNoElim      = "vote.noelim"
	cueEndMafia        = "end.mafia"
	cueEndCitizen      = "end.citizen"
	cueEndForce        = "end.force"
	cueVoiceOn         = "voice.on"
	cueVoiceOff        = "voice.off"
	cueGamePaused      = "game.paused"
	cueGameResumed     = "game.resumed"
	cueSystemRestore   = "system.restore"
	cueSystemPersistFailure = "system.persist.failure"
)

// defaultCatalog is the FR-7.2 default Korean catalog implementation.
// It is stateless and safe for concurrent use.
type defaultCatalog struct{}

// NewDefaultCatalog returns the bundled Korean catalog.
func NewDefaultCatalog() AnnouncementCatalog { return defaultCatalog{} }

// Render implements AnnouncementCatalog.
func (defaultCatalog) Render(env game.EventEnvelope, ctx CatalogContext) Announcement {
	switch e := env.Event.(type) {
	case game.GameStarted:
		return ann(msgGameStarted, cueGameStarted, SeverityEmphasis)

	case game.PhaseChanged:
		switch e.Phase {
		case game.PhaseIntro:
			seconds := ctx.IntroSecondsPerPlayer
			if seconds <= 0 {
				seconds = 20
			}
			return ann(fmt.Sprintf(msgPhaseIntro, seconds), cuePhaseIntro, SeverityInfo)
		case game.PhaseNight:
			return ann(msgPhaseNight, cuePhaseNight, SeverityEmphasis)
		case game.PhaseDay:
			// Day 1 follows INTRO directly — no preceding night, no
			// DeathAnnounced / PeacefulNight will be emitted, so we use a
			// dedicated subtitle that doesn't reference last night.
			if e.Day == 1 {
				return ann(msgPhaseDayFirst, cuePhaseDayFirst, SeverityEmphasis)
			}
			return ann(fmt.Sprintf(msgPhaseDay, e.Day), cuePhaseDay, SeverityEmphasis)
		case game.PhaseVote:
			return ann(msgPhaseVote, cuePhaseVote, SeverityEmphasis)
		case game.PhaseRecount:
			return ann(msgPhaseRecount, cuePhaseRecount, SeverityWarn)
		}
		return Announcement{}

	case game.NightStepChanged:
		switch e.Step {
		case game.NightStepIntro:
			// Iteration 8: PhaseChanged{NIGHT} 의 phase.night cue 가 안내를
			// 담당하므로 INTRO 단계는 별도 cue 를 발화하지 않는다.
			return Announcement{}
		case game.NightStepMafia:
			return ann(msgNightStepMafia, cueNightMafia, SeverityEmphasis)
		case game.NightStepPolice:
			return ann(msgNightStepPolice, cueNightPolice, SeverityEmphasis)
		case game.NightStepDoctor:
			return ann(msgNightStepDoctor, cueNightDoctor, SeverityEmphasis)
		}
		return Announcement{}

	case game.IntroSpeakerChanged:
		return ann(fmt.Sprintf(msgIntroSpeaker, ctx.nameOf(e.PlayerID)), cueIntroSpeaker, SeverityInfo)

	case game.DeathAnnounced:
		return ann(fmt.Sprintf(msgDeath, ctx.nameOf(e.Victim)), cueDeathAnnounced, SeverityEmphasis)

	case game.PeacefulNight:
		return ann(msgPeacefulNight, cuePeacefulNight, SeverityInfo)

	case game.Eliminated:
		// Iter7 §3.5 — voice splits on mafia vs non-mafia; subtitle keeps
		// the exact role-kr so players still see which role was lynched.
		cue := cueEliminatedNotMafia
		if e.Role == game.RoleMafia {
			cue = cueEliminatedMafia
		}
		return ann(fmt.Sprintf(msgEliminated, ctx.nameOf(e.PlayerID), roleKr(e.Role)), cue, SeverityEmphasis)

	case game.DiscussionTimerTick:
		switch e.SecondsLeft {
		case 30:
			return ann(msgTimer30, cueTimer30, SeverityInfo)
		case 10:
			return ann(msgTimer10, cueTimer10, SeverityWarn)
		case 0:
			return ann(msgTimer0, cueTimer0, SeverityInfo)
		}
		return Announcement{}

	case game.VoteTallied:
		switch {
		case e.Recount:
			// Suppressed: a follow-up PhaseChanged{PhaseRecount} carries
			// the recount narration. Emitting both produced two near-
			// identical Korean lines back-to-back during RECOUNT entry.
			return Announcement{}
		case e.Eliminated == nil:
			return ann(msgVoteNoElim, cueVoteNoElim, SeverityInfo)
		default:
			// Suppressed: a follow-up Eliminated event carries the speech.
			return Announcement{}
		}

	case game.GameEnded:
		switch e.EndReason {
		case game.EndMafiaWin:
			return ann(msgEndMafia, cueEndMafia, SeverityEmphasis)
		case game.EndCitizenWin:
			return ann(msgEndCitizen, cueEndCitizen, SeverityEmphasis)
		case game.EndHostForceEnd:
			return ann(msgEndForce, cueEndForce, SeverityInfo)
		}
		return Announcement{}

	case game.VoiceToggled:
		if e.On {
			return ann(msgVoiceOn, cueVoiceOn, SeverityInfo)
		}
		return ann(msgVoiceOff, cueVoiceOff, SeverityInfo)

	case game.GamePaused:
		return ann(msgGamePaused, cueGamePaused, SeverityInfo)

	case game.GameResumed:
		return ann(msgGameResumed, cueGameResumed, SeverityInfo)

	// Private events (RoleRevealedToPlayer, MafiaCohortRevealed,
	// MafiaTargetSelected, PoliceResult, MafiaRepresentativeReassigned)
	// produce no announcement (BR-U2-CAT-1).
	default:
		return Announcement{}
	}
}

// ann constructs a fully populated public-facing Announcement. Subtitle
// keeps dynamic interpolation; AudioID names a stable cue (Iter7 FR-8.9).
func ann(text, audioID string, sev Severity) Announcement {
	return Announcement{
		Subtitle:      text,
		AudioID:       audioID,
		Severity:      sev,
		ForPublicOnly: true,
	}
}

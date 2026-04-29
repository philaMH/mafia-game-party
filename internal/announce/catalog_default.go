package announce

import (
	"fmt"

	"github.com/saltware/mafia-game/internal/game"
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
		return ann(msgGameStarted, SeverityEmphasis)

	case game.PhaseChanged:
		switch e.Phase {
		case game.PhaseIntro:
			seconds := ctx.IntroSecondsPerPlayer
			if seconds <= 0 {
				seconds = 20
			}
			return ann(fmt.Sprintf(msgPhaseIntro, seconds), SeverityInfo)
		case game.PhaseNight:
			return ann(msgPhaseNight, SeverityEmphasis)
		case game.PhaseDay:
			// Day 1 follows INTRO directly — no preceding night, no
			// DeathAnnounced / PeacefulNight will be emitted, so we use a
			// dedicated subtitle that doesn't reference last night.
			if e.Day == 1 {
				return ann(msgPhaseDayFirst, SeverityEmphasis)
			}
			return ann(fmt.Sprintf(msgPhaseDay, e.Day), SeverityEmphasis)
		case game.PhaseVote:
			return ann(msgPhaseVote, SeverityEmphasis)
		case game.PhaseRecount:
			return ann(msgPhaseRecount, SeverityWarn)
		}
		return Announcement{}

	case game.NightStepChanged:
		switch e.Step {
		case game.NightStepMafia:
			return ann(msgNightStepMafia, SeverityEmphasis)
		case game.NightStepPolice:
			return ann(msgNightStepPolice, SeverityEmphasis)
		case game.NightStepDoctor:
			return ann(msgNightStepDoctor, SeverityEmphasis)
		}
		return Announcement{}

	case game.IntroSpeakerChanged:
		return ann(fmt.Sprintf(msgIntroSpeaker, ctx.nameOf(e.PlayerID)), SeverityInfo)

	case game.DeathAnnounced:
		return ann(fmt.Sprintf(msgDeath, ctx.nameOf(e.Victim)), SeverityEmphasis)

	case game.PeacefulNight:
		return ann(msgPeacefulNight, SeverityInfo)

	case game.Eliminated:
		return ann(fmt.Sprintf(msgEliminated, ctx.nameOf(e.PlayerID), roleKr(e.Role)), SeverityEmphasis)

	case game.DiscussionTimerTick:
		switch e.SecondsLeft {
		case 30:
			return ann(msgTimer30, SeverityInfo)
		case 10:
			return ann(msgTimer10, SeverityWarn)
		case 0:
			return ann(msgTimer0, SeverityInfo)
		}
		return Announcement{}

	case game.VoteTallied:
		switch {
		case e.Recount:
			return ann(msgVoteRecount, SeverityWarn)
		case e.Eliminated == nil:
			return ann(msgVoteNoElim, SeverityInfo)
		default:
			// Suppressed: a follow-up Eliminated event carries the speech.
			return Announcement{}
		}

	case game.GameEnded:
		switch e.EndReason {
		case game.EndMafiaWin:
			return ann(msgEndMafia, SeverityEmphasis)
		case game.EndCitizenWin:
			return ann(msgEndCitizen, SeverityEmphasis)
		case game.EndHostForceEnd:
			return ann(msgEndForce, SeverityInfo)
		}
		return Announcement{}

	case game.VoiceToggled:
		if e.On {
			return ann(msgVoiceOn, SeverityInfo)
		}
		return ann(msgVoiceOff, SeverityInfo)

	case game.GamePaused:
		return ann(msgGamePaused, SeverityInfo)

	case game.GameResumed:
		return ann(msgGameResumed, SeverityInfo)

	// Private events (RoleRevealedToPlayer, MafiaCohortRevealed,
	// MafiaTargetSelected, PoliceResult, MafiaRepresentativeReassigned)
	// produce no announcement (BR-U2-CAT-1).
	default:
		return Announcement{}
	}
}

// ann constructs a fully populated Announcement (Subtitle == Speech,
// ForPublicOnly = true per BR-U2-CAT-8).
func ann(text string, sev Severity) Announcement {
	return Announcement{
		Subtitle:      text,
		Speech:        text,
		Severity:      sev,
		ForPublicOnly: true,
	}
}

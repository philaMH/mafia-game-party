package session

import "github.com/saltware/mafia-game/internal/game"

// PrivateView is the masked snapshot delivered to a single viewer.
// Caller decides which viewer to build for via BuildPrivateView's pid arg.
//
// Masking rules (NFR-U2-S4, BR-U2-MASK-*):
//   - PublicView (pid == ""):           all Role/Keyword stripped
//   - PlayerView (others):              Role/Keyword stripped
//   - PlayerView (self):                own Role/Keyword visible
//   - PlayerView (mafia → other mafia): Role visible; Keyword still hidden
//   - PhaseEnd:                         all roles revealed (Reveal)
type PrivateView struct {
	State       game.State
	YourRole    game.Role
	YourKeyword string
	YourTeam    game.Team
	MafiaCohort []game.PlayerID
	IsHost      bool
}

// BuildPrivateView returns a fresh PrivateView for viewer pid against the
// given snapshot. Pass an empty pid to build a PublicView (no self-data).
func BuildPrivateView(state game.State, pid game.PlayerID, hostID game.PlayerID) PrivateView {
	view := state.Clone()

	// Capture the viewer's own role from the original (pre-mask) state so
	// we can restore it after the blanket scrub below.
	var (
		myRole    game.Role
		myKeyword string
		myTeam    game.Team
		isMafia   bool
		isPolice  bool
	)
	if pid != "" {
		if me, ok := state.FindPlayer(pid); ok {
			myRole = me.Role
			myKeyword = me.Keyword
			myTeam = game.TeamOf(me.Role)
			isMafia = me.Role == game.RoleMafia
			isPolice = me.Role == game.RolePolice
		}
	}

	// PoliceHistory is private to the police officer. Scrub it for everyone
	// else (PUBLIC viewer included). The role-revealing END phase still
	// hides the history because only the police "earned" the information.
	if !isPolice {
		view.PoliceHistory = nil
	}

	// Phase-END reveal: leave Roles in place; only Keyword is scrubbed for
	// non-self viewers.
	revealAll := state.Phase == game.PhaseEnd

	for i := range view.Players {
		p := &view.Players[i]
		// Default scrub.
		if !revealAll {
			p.Role = ""
		}
		// Keyword always hidden for others (even after END reveal).
		if p.ID != pid {
			p.Keyword = ""
		}
	}

	// Re-add self info.
	if pid != "" {
		for i := range view.Players {
			p := &view.Players[i]
			if p.ID == pid {
				p.Role = myRole
				p.Keyword = myKeyword
			}
		}
	}

	// Mafia ally Role disclosure (Keyword still hidden).
	var cohort []game.PlayerID
	if isMafia {
		cohort = make([]game.PlayerID, 0, 3)
		for i := range view.Players {
			p := &view.Players[i]
			// Look up the original role (state, not view) for accuracy.
			orig, ok := state.FindPlayer(p.ID)
			if !ok {
				continue
			}
			if orig.Role == game.RoleMafia {
				p.Role = game.RoleMafia
				cohort = append(cohort, p.ID)
			}
		}
	}

	return PrivateView{
		State:       view,
		YourRole:    myRole,
		YourKeyword: myKeyword,
		YourTeam:    myTeam,
		MafiaCohort: cohort,
		IsHost:      pid == hostID && pid != "",
	}
}

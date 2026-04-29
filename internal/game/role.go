package game

import "math/rand"

// Assignments is the result of RoleAssigner.Assign: per-player role &
// keyword maps plus the chosen mafia representative.
type Assignments struct {
	PlayerRoles      map[PlayerID]Role
	PlayerKeywords   map[PlayerID]string
	MafiaIDs         []PlayerID
	RepresentativeID PlayerID
}

// RoleAssigner shuffles the participant list, assigns roles based on the
// game options, and selects the mafia representative.
type RoleAssigner interface {
	Assign(playerIDs []PlayerID, opts Options, rng *rand.Rand) (Assignments, error)
}

// NewAssigner returns the default RoleAssigner backed by the supplied
// keyword pool. All four role pools must be populated.
func NewAssigner(pool KeywordPool) RoleAssigner {
	return &defaultAssigner{pool: pool}
}

type defaultAssigner struct {
	pool KeywordPool
}

// Assign implements RoleAssigner. Algorithm (see business-logic-model.md §9):
//
//  1. Validate player count and options.
//  2. Shuffle the player IDs.
//  3. Take MafiaCount as MAFIA, then 1 as DOCTOR, then 1 as POLICE, rest as
//     CITIZEN.
//  4. Pick one keyword per role; all players sharing a role share the keyword
//     (Q-FD-U1-8=A).
//  5. Pick a mafia representative uniformly at random from the mafia subset
//     (Q-FD-U1-4-FU=C); when MafiaCount == 1, that lone mafia is the
//     representative.
func (a *defaultAssigner) Assign(playerIDs []PlayerID, opts Options, rng *rand.Rand) (Assignments, error) {
	n := len(playerIDs)
	if err := validateOptions(opts, n); err != nil {
		return Assignments{}, err
	}

	shuffled := make([]PlayerID, n)
	copy(shuffled, playerIDs)
	rng.Shuffle(n, func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	roles := make(map[PlayerID]Role, n)
	mafiaIDs := make([]PlayerID, 0, opts.MafiaCount)

	idx := 0
	for ; idx < opts.MafiaCount; idx++ {
		roles[shuffled[idx]] = RoleMafia
		mafiaIDs = append(mafiaIDs, shuffled[idx])
	}
	roles[shuffled[idx]] = RoleDoctor
	idx++
	roles[shuffled[idx]] = RolePolice
	idx++
	for ; idx < n; idx++ {
		roles[shuffled[idx]] = RoleCitizen
	}

	keywords := make(map[PlayerID]string, n)
	mafiaKw, err := a.pool.Pick(RoleMafia, rng)
	if err != nil {
		return Assignments{}, err
	}
	citizenKw, err := a.pool.Pick(RoleCitizen, rng)
	if err != nil {
		return Assignments{}, err
	}
	doctorKw, err := a.pool.Pick(RoleDoctor, rng)
	if err != nil {
		return Assignments{}, err
	}
	policeKw, err := a.pool.Pick(RolePolice, rng)
	if err != nil {
		return Assignments{}, err
	}
	for pid, r := range roles {
		switch r {
		case RoleMafia:
			keywords[pid] = mafiaKw
		case RoleCitizen:
			keywords[pid] = citizenKw
		case RoleDoctor:
			keywords[pid] = doctorKw
		case RolePolice:
			keywords[pid] = policeKw
		}
	}

	rep := mafiaIDs[0]
	if len(mafiaIDs) > 1 {
		rep = mafiaIDs[rng.Intn(len(mafiaIDs))]
	}

	return Assignments{
		PlayerRoles:      roles,
		PlayerKeywords:   keywords,
		MafiaIDs:         mafiaIDs,
		RepresentativeID: rep,
	}, nil
}

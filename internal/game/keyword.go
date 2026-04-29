package game

import "math/rand"

// KeywordPool returns one keyword per role for a single game. It is the
// extension seam called out by FR-7.1: callers may supply external
// implementations (e.g., loaded from a YAML/JSON file) without changing the
// engine.
//
// Pick is called once per role per game by RoleAssigner; players sharing a
// role share the chosen keyword (per Q-FD-U1-8=A).
type KeywordPool interface {
	Pick(role Role, rng *rand.Rand) (string, error)
}

// mapKeywordPool is the default in-memory implementation backed by four
// per-role string slices. Used both by NewDefaultKeywordPool and
// LoadKeywordPool.
type mapKeywordPool struct {
	mafia   []string
	citizen []string
	doctor  []string
	police  []string
}

// Pick selects one keyword for the given role. It returns an error if the
// pool for that role is empty.
func (p mapKeywordPool) Pick(role Role, rng *rand.Rand) (string, error) {
	pool := p.poolFor(role)
	if len(pool) == 0 {
		return "", errf(CodeValidation, "keyword pool for role %s is empty", role)
	}
	idx := rng.Intn(len(pool))
	return pool[idx], nil
}

// poolFor returns the slice for the given role.
func (p mapKeywordPool) poolFor(role Role) []string {
	switch role {
	case RoleMafia:
		return p.mafia
	case RoleCitizen:
		return p.citizen
	case RoleDoctor:
		return p.doctor
	case RolePolice:
		return p.police
	default:
		return nil
	}
}

// NewDefaultKeywordPool returns a pool backed by the package-level Korean
// defaults (see keyword_pool_data.go).
func NewDefaultKeywordPool() KeywordPool {
	return mapKeywordPool{
		mafia:   defaultMafiaWords,
		citizen: defaultCitizenWords,
		doctor:  defaultDoctorWords,
		police:  defaultPoliceWords,
	}
}

package game

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"testing"
)

func TestDefaultKeywordPools_NoDuplicatesPerRole(t *testing.T) {
	for _, c := range []struct {
		name string
		pool []string
		min  int
	}{
		{"mafia", defaultMafiaWords, 30},
		{"citizen", defaultCitizenWords, 30},
		{"doctor", defaultDoctorWords, 25},
		{"police", defaultPoliceWords, 25},
	} {
		seen := map[string]bool{}
		for _, w := range c.pool {
			if w == "" {
				t.Errorf("%s: empty entry", c.name)
			}
			if seen[w] {
				t.Errorf("%s: duplicate %q", c.name, w)
			}
			seen[w] = true
		}
		if len(c.pool) < c.min {
			t.Errorf("%s: pool size %d < min %d", c.name, len(c.pool), c.min)
		}
	}
}

func TestDefaultKeywordPool_PicksFromCorrectRole(t *testing.T) {
	pool := NewDefaultKeywordPool()
	rng := rand.New(rand.NewSource(1))
	w, err := pool.Pick(RoleMafia, rng)
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	found := false
	for _, m := range defaultMafiaWords {
		if m == w {
			found = true
		}
	}
	if !found {
		t.Errorf("picked %q is not in mafia pool", w)
	}
}

func TestLoadKeywordPool_RoundTrip(t *testing.T) {
	doc := keywordPoolJSON{
		Mafia:   []string{"a"},
		Citizen: []string{"b"},
		Doctor:  []string{"c"},
		Police:  []string{"d"},
	}
	b, _ := json.Marshal(doc)
	pool, err := LoadKeywordPool(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("LoadKeywordPool: %v", err)
	}
	rng := rand.New(rand.NewSource(0))
	for r, want := range map[Role]string{
		RoleMafia: "a", RoleCitizen: "b", RoleDoctor: "c", RolePolice: "d",
	} {
		got, err := pool.Pick(r, rng)
		if err != nil {
			t.Fatalf("Pick(%s): %v", r, err)
		}
		if got != want {
			t.Errorf("Pick(%s)=%q, want %q", r, got, want)
		}
	}
}

func TestLoadKeywordPool_RejectsEmptyRole(t *testing.T) {
	doc := keywordPoolJSON{
		Mafia:   []string{},
		Citizen: []string{"a"},
		Doctor:  []string{"a"},
		Police:  []string{"a"},
	}
	b, _ := json.Marshal(doc)
	if _, err := LoadKeywordPool(bytes.NewReader(b)); err == nil {
		t.Errorf("expected error for empty mafia pool")
	}
}

func TestKeywordPool_EmptyPickError(t *testing.T) {
	p := mapKeywordPool{}
	rng := rand.New(rand.NewSource(0))
	if _, err := p.Pick(RoleMafia, rng); err == nil {
		t.Errorf("expected error for empty pool")
	}
}

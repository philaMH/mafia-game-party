package game

import (
	"encoding/json"
	"io"
)

// keywordPoolJSON is the on-disk JSON shape used by LoadKeywordPool.
// It is intentionally simple so operators can hand-edit the file.
type keywordPoolJSON struct {
	Mafia   []string `json:"mafia"`
	Citizen []string `json:"citizen"`
	Doctor  []string `json:"doctor"`
	Police  []string `json:"police"`
}

// LoadKeywordPool reads a JSON document from r and returns a KeywordPool
// backed by the four per-role slices it contains. The expected JSON shape:
//
//	{ "mafia": ["..."], "citizen": ["..."], "doctor": ["..."], "police": ["..."] }
//
// FR-7.1: this is the extension seam used to swap out the built-in Korean
// pool without recompiling the binary.
func LoadKeywordPool(r io.Reader) (KeywordPool, error) {
	var raw keywordPoolJSON
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, errf(CodeValidation, "decode keyword pool: %v", err)
	}
	if len(raw.Mafia) == 0 {
		return nil, errf(CodeValidation, "mafia keyword pool must be non-empty")
	}
	if len(raw.Citizen) == 0 {
		return nil, errf(CodeValidation, "citizen keyword pool must be non-empty")
	}
	if len(raw.Doctor) == 0 {
		return nil, errf(CodeValidation, "doctor keyword pool must be non-empty")
	}
	if len(raw.Police) == 0 {
		return nil, errf(CodeValidation, "police keyword pool must be non-empty")
	}
	return mapKeywordPool{
		mafia:   raw.Mafia,
		citizen: raw.Citizen,
		doctor:  raw.Doctor,
		police:  raw.Police,
	}, nil
}

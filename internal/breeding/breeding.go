// Package breeding resolves Palworld breeding outcomes from bundled game-rule data.
package breeding

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed breeding-rules.json
var bundledRules []byte

type dataFile struct {
	Records []record `json:"records"`
	Combos  []combo  `json:"combos"`
}

type record struct {
	ID            string `json:"id"`
	Key           string `json:"key"`
	Rank          int    `json:"rank"`
	Priority      int    `json:"priority"`
	IgnoreGeneric bool   `json:"ignoreGeneric"`
	Boss          bool   `json:"boss"`
}

type combo struct {
	A     string `json:"a"`
	B     string `json:"b"`
	Child string `json:"child"`
	GA    string `json:"ga"`
	GB    string `json:"gb"`
}

// Result is the exact offspring selected by the bundled rule set.
type Result struct {
	Child       string
	Rule        string
	ParentARank int
	ParentBRank int
	TargetRank  int
}

// Rules is an immutable indexed breeding-rule dataset.
type Rules struct {
	byID         map[string]record
	generic      []record
	combosByPair map[string][]combo
	maxRank      int
}

// Default loads the versioned rules bundled with this executable.
func Default() (*Rules, error) {
	var data dataFile
	if err := json.Unmarshal(bundledRules, &data); err != nil {
		return nil, fmt.Errorf("decode bundled breeding rules: %w", err)
	}
	return newRules(data)
}

func newRules(data dataFile) (*Rules, error) {
	rules := &Rules{byID: make(map[string]record, len(data.Records)), combosByPair: make(map[string][]combo), maxRank: 0}
	specialChildren := make(map[string]struct{})
	for _, entry := range data.Combos {
		specialChildren[entry.Child] = struct{}{}
	}
	for _, entry := range data.Records {
		entry.ID = canonical(entry.ID)
		entry.Key = strings.TrimSpace(entry.Key)
		if entry.ID == "" || entry.Key == "" || entry.Rank < 0 || entry.Priority < 0 {
			return nil, fmt.Errorf("invalid breeding record %q", entry.Key)
		}
		if _, exists := rules.byID[entry.ID]; exists {
			return nil, fmt.Errorf("duplicate breeding record %q", entry.ID)
		}
		rules.byID[entry.ID] = entry
		if entry.Rank > rules.maxRank {
			rules.maxRank = entry.Rank
		}
	}
	for _, entry := range data.Combos {
		if _, ok := rules.byID[canonical(entry.A)]; !ok {
			return nil, fmt.Errorf("special combination references unknown Pal %q", entry.A)
		}
		if _, ok := rules.byID[canonical(entry.B)]; !ok {
			return nil, fmt.Errorf("special combination references unknown Pal %q", entry.B)
		}
		if _, ok := rules.byID[canonical(entry.Child)]; !ok {
			return nil, fmt.Errorf("special combination references unknown child %q", entry.Child)
		}
		key := pairKey(canonical(entry.A), canonical(entry.B))
		rules.combosByPair[key] = append(rules.combosByPair[key], entry)
		reverseKey := pairKey(canonical(entry.B), canonical(entry.A))
		if reverseKey != key {
			rules.combosByPair[reverseKey] = append(rules.combosByPair[reverseKey], entry)
		}
	}
	for _, entry := range rules.byID {
		if !entry.Boss && !entry.IgnoreGeneric {
			if _, special := specialChildren[entry.ID]; !special {
				rules.generic = append(rules.generic, entry)
			}
		}
	}
	if len(rules.generic) == 0 {
		return nil, fmt.Errorf("no generic breeding candidates")
	}
	sort.Slice(rules.generic, func(i, j int) bool {
		if rules.generic[i].Rank != rules.generic[j].Rank {
			return rules.generic[i].Rank < rules.generic[j].Rank
		}
		return rules.generic[i].Priority > rules.generic[j].Priority
	})
	return rules, nil
}

// Resolve returns an offspring only when both Character IDs are recognized.
func (rules *Rules) Resolve(parentA, genderA, parentB, genderB string) (Result, bool) {
	a, okA := rules.byID[canonical(parentA)]
	b, okB := rules.byID[canonical(parentB)]
	if !okA || !okB {
		return Result{}, false
	}
	genderA, genderB = genderCode(genderA), genderCode(genderB)
	for _, special := range rules.combosByPair[pairKey(a.ID, b.ID)] {
		if comboMatches(special, a.ID, genderA, b.ID, genderB) || comboMatches(special, b.ID, genderB, a.ID, genderA) {
			return Result{Child: rules.byID[canonical(special.Child)].Key, Rule: "special", ParentARank: a.Rank, ParentBRank: b.Rank}, true
		}
	}
	if a.ID == b.ID {
		return Result{Child: a.Key, Rule: "same_species", ParentARank: a.Rank, ParentBRank: b.Rank, TargetRank: a.Rank}, true
	}
	target := (a.Rank + b.Rank + 1) / 2
	chosen := rules.generic[0]
	bestDistance := abs(chosen.Rank - target)
	for _, candidate := range rules.generic[1:] {
		distance := abs(candidate.Rank - target)
		if distance < bestDistance || (distance == bestDistance && candidate.Priority > chosen.Priority) {
			chosen, bestDistance = candidate, distance
		}
	}
	return Result{Child: chosen.Key, Rule: "generic", ParentARank: a.Rank, ParentBRank: b.Rank, TargetRank: target}, true
}

func pairKey(left, right string) string { return left + "\x00" + right }

// Key returns the canonical Character ID used by breeding outcomes.
func (rules *Rules) Key(value string) (string, bool) {
	entry, ok := rules.byID[canonical(value)]
	if !ok {
		return "", false
	}
	return entry.Key, true
}

// Species returns every Pal Character ID in the bundled breeding dataset.
func (rules *Rules) Species() []string {
	values := make([]string, 0, len(rules.byID))
	for _, entry := range rules.byID {
		values = append(values, entry.Key)
	}
	sort.Strings(values)
	return values
}

func comboMatches(special combo, a, genderA, b, genderB string) bool {
	if canonical(special.A) != a || canonical(special.B) != b {
		return false
	}
	return (special.GA == "" || special.GA == genderA) && (special.GB == "" || special.GB == genderB)
}

func canonical(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.TrimPrefix(value, "boss_")
}

func genderCode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "male", "m":
		return "M"
	case "female", "f":
		return "F"
	default:
		return ""
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// Package planner builds local breeding plans from a player's current collection.
package planner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Ltorre/palpedia-snapshot/internal/breeding"
)

// Pal is one player-owned Pal available to the breeding planner.
type Pal struct {
	InstanceID  string
	CharacterID string
	DisplayName string
	Gender      string
	Level       int32
	Traits      []string
}

// SortOrder controls the order used when browsing the collection.
type SortOrder int

const (
	SortByName SortOrder = iota
	SortByLevelAscending
	SortByLevelDescending
)

// FilterOptions controls collection search, sex filters, and ordering.
type FilterOptions struct {
	Query          string
	GoldOnly       bool
	DiamondOnly    bool
	MaleOnly       bool
	FemaleOnly     bool
	RequiredGender string
	SortOrder      SortOrder
	Deduplicate    bool
}

// PalName returns a player-facing name when it is available.
func PalName(pal Pal) string {
	if strings.TrimSpace(pal.DisplayName) != "" {
		return pal.DisplayName
	}
	return pal.CharacterID
}

// PalLevel returns the minimum in-game level when a save omits its default level.
func PalLevel(pal Pal) int32 {
	if pal.Level < 1 {
		return 1
	}
	return pal.Level
}

// PairResult describes the exact outcome for two selected Pals.
type PairResult struct {
	Child      string
	Rule       string
	TargetRank int
}

// ResolvePair calculates one real male/female pair from the loaded collection.
func ResolvePair(rules *breeding.Rules, male, female Pal) (PairResult, error) {
	if !strings.EqualFold(male.Gender, "male") || !strings.EqualFold(female.Gender, "female") {
		return PairResult{}, fmt.Errorf("choose one male parent and one female parent")
	}
	result, ok := rules.Resolve(male.CharacterID, male.Gender, female.CharacterID, female.Gender)
	if !ok {
		return PairResult{}, fmt.Errorf("breeding data is unavailable for %q or %q", male.CharacterID, female.CharacterID)
	}
	return PairResult{Child: result.Child, Rule: result.Rule, TargetRank: result.TargetRank}, nil
}

// Filter returns Pals that match text and at least one requested high-tier group.
func Filter(pals []Pal, query string, goldOnly, diamondOnly bool) []Pal {
	return FilterWithOptions(pals, FilterOptions{Query: query, GoldOnly: goldOnly, DiamondOnly: diamondOnly})
}

// FilterWithOptions returns Pals matching the selected browse controls.
func FilterWithOptions(pals []Pal, options FilterOptions) []Pal {
	query := strings.ToLower(strings.TrimSpace(options.Query))
	requiredGender := strings.ToLower(strings.TrimSpace(options.RequiredGender))
	filtered := make([]Pal, 0, len(pals))
	for _, pal := range pals {
		gender := strings.ToLower(pal.Gender)
		if requiredGender != "" && gender != requiredGender {
			continue
		}
		if requiredGender == "" && options.MaleOnly != options.FemaleOnly {
			if options.MaleOnly && gender != "male" {
				continue
			}
			if options.FemaleOnly && gender != "female" {
				continue
			}
		}
		if options.GoldOnly || options.DiamondOnly {
			match := false
			for _, trait := range pal.Traits {
				match = match || (options.GoldOnly && Tier(trait) == Gold) || (options.DiamondOnly && Tier(trait) == Diamond)
			}
			if !match {
				continue
			}
		}
		if query != "" && !matches(pal, query) {
			continue
		}
		filtered = append(filtered, pal)
	}
	if options.Deduplicate {
		filtered = highestLevelUnique(filtered)
	}
	sort.Slice(filtered, func(i, j int) bool {
		left, right := PalLevel(filtered[i]), PalLevel(filtered[j])
		if options.SortOrder == SortByLevelAscending && left != right {
			return left < right
		}
		if options.SortOrder == SortByLevelDescending && left != right {
			return left > right
		}
		return strings.Join([]string{PalName(filtered[i]), filtered[i].Gender, filtered[i].InstanceID}, "\x00") < strings.Join([]string{PalName(filtered[j]), filtered[j].Gender, filtered[j].InstanceID}, "\x00")
	})
	return filtered
}

func highestLevelUnique(pals []Pal) []Pal {
	best := make(map[string]Pal, len(pals))
	for _, pal := range pals {
		key := strings.Join([]string{strings.ToLower(pal.CharacterID), strings.ToLower(pal.Gender), traitSetKey(pal.Traits)}, "\x00")
		current, exists := best[key]
		if !exists || PalLevel(pal) > PalLevel(current) || (PalLevel(pal) == PalLevel(current) && pal.InstanceID < current.InstanceID) {
			best[key] = pal
		}
	}
	unique := make([]Pal, 0, len(best))
	for _, pal := range best {
		unique = append(unique, pal)
	}
	return unique
}

func traitSetKey(traits []string) string {
	values := make([]string, 0, len(traits))
	for _, trait := range traits {
		values = append(values, strings.ToLower(strings.TrimSpace(trait)))
	}
	sort.Strings(values)
	return strings.Join(values, "\x00")
}

func matches(pal Pal, query string) bool {
	if strings.Contains(strings.ToLower(PalName(pal)), query) || strings.Contains(strings.ToLower(pal.CharacterID), query) || strings.Contains(strings.ToLower(pal.Gender), query) {
		return true
	}
	for _, trait := range pal.Traits {
		if strings.Contains(strings.ToLower(trait), query) || strings.Contains(strings.ToLower(TraitName(trait)), query) {
			return true
		}
	}
	return false
}

// Step is one breeding operation in a fastest sequential-generation route.
type Step struct {
	Generation int
	ParentA    string
	ParentB    string
	Child      string
	Rule       string
}

// Path is a reproducible species route. It does not model passive inheritance or egg RNG.
type Path struct {
	Target      string
	Generations int
	Steps       []Step
}

// ShortestPath finds the fewest sequential breeding generations from the collection to target.
// It assumes an offspring can be bred again as the needed sex; actual egg genders can require retries.
func ShortestPath(rules *breeding.Rules, pals []Pal, target string) (Path, error) {
	return shortestPath(rules, pals, target, nil)
}

// ShortestPathAvoidingSpecies finds a route that never uses the listed species
// as a starting parent or a generated intermediate Pal. The target itself is
// always allowed so callers can request a route to it.
func ShortestPathAvoidingSpecies(rules *breeding.Rules, pals []Pal, target string, excluded map[string]bool) (Path, error) {
	targetKey, ok := rules.Key(target)
	if !ok {
		return Path{}, fmt.Errorf("unknown target Pal %q", target)
	}
	allowed := make(map[string]bool, len(excluded))
	for species := range excluded {
		if key, known := rules.Key(species); known && key != targetKey {
			allowed[key] = true
		}
	}
	return shortestPath(rules, pals, targetKey, allowed)
}

func shortestPath(rules *breeding.Rules, pals []Pal, target string, excluded map[string]bool) (Path, error) {
	target, ok := rules.Key(target)
	if !ok {
		return Path{}, fmt.Errorf("unknown target Pal %q", target)
	}
	species := rules.Species()
	type state struct {
		generation int
		step       *Step
	}
	const unreachable = int(^uint(0) >> 1)
	states := make(map[string][2]state, len(species))
	for _, name := range species {
		states[name] = [2]state{{generation: unreachable}, {generation: unreachable}}
	}
	for _, pal := range pals {
		name, known := rules.Key(pal.CharacterID)
		if !known || excluded[name] {
			continue
		}
		entry := states[name]
		switch strings.ToLower(pal.Gender) {
		case "male":
			entry[0] = state{generation: 0}
		case "female":
			entry[1] = state{generation: 0}
		}
		states[name] = entry
	}
	for iteration := 0; iteration < len(species); iteration++ {
		changed := false
		for _, male := range species {
			if excluded[male] {
				continue
			}
			maleState := states[male][0]
			if maleState.generation == unreachable {
				continue
			}
			for _, female := range species {
				if excluded[female] {
					continue
				}
				femaleState := states[female][1]
				if femaleState.generation == unreachable {
					continue
				}
				result, resolved := rules.Resolve(male, "male", female, "female")
				if !resolved {
					continue
				}
				child := result.Child
				if excluded[child] {
					continue
				}
				generation := max(maleState.generation, femaleState.generation) + 1
				step := &Step{Generation: generation, ParentA: male, ParentB: female, Child: child, Rule: result.Rule}
				entry := states[child]
				for gender := range entry {
					if generation < entry[gender].generation || (generation == entry[gender].generation && stepKey(step) < stepKey(entry[gender].step)) {
						entry[gender] = state{generation: generation, step: step}
						changed = true
					}
				}
				states[child] = entry
			}
		}
		if !changed {
			break
		}
	}
	targetState := states[target][0]
	if states[target][1].generation < targetState.generation {
		targetState = states[target][1]
	}
	if targetState.generation == unreachable {
		return Path{}, fmt.Errorf("no breeding route from the loaded male/female collection to %s", target)
	}
	path := Path{Target: target, Generations: targetState.generation}
	seen := make(map[string]bool)
	var add func(*Step)
	add = func(step *Step) {
		if step == nil {
			return
		}
		key := stepKey(step)
		if seen[key] {
			return
		}
		seen[key] = true
		add(states[step.ParentA][0].step)
		add(states[step.ParentB][1].step)
		path.Steps = append(path.Steps, *step)
	}
	add(targetState.step)
	return path, nil
}

// ShortestPathAsIfUnowned finds a route after removing already-owned copies of
// the target from the starting collection. It is useful when planning passive
// inheritance into a species the player already owns.
func ShortestPathAsIfUnowned(rules *breeding.Rules, pals []Pal, target string) (Path, error) {
	return ShortestPathAsIfUnownedAvoidingSpecies(rules, pals, target, nil)
}

// ShortestPathAsIfUnownedAvoidingSpecies removes existing copies of the target
// from the starting collection while also avoiding the selected other species.
func ShortestPathAsIfUnownedAvoidingSpecies(rules *breeding.Rules, pals []Pal, target string, excluded map[string]bool) (Path, error) {
	targetKey, ok := rules.Key(target)
	if !ok {
		return Path{}, fmt.Errorf("unknown target Pal %q", target)
	}
	withoutTarget := make([]Pal, 0, len(pals))
	for _, pal := range pals {
		palKey, known := rules.Key(pal.CharacterID)
		if known && palKey == targetKey {
			continue
		}
		withoutTarget = append(withoutTarget, pal)
	}
	return ShortestPathAvoidingSpecies(rules, withoutTarget, targetKey, excluded)
}

func stepKey(step *Step) string {
	if step == nil {
		return ""
	}
	return strings.Join([]string{step.ParentA, step.ParentB, step.Child, step.Rule}, "\x00")
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}

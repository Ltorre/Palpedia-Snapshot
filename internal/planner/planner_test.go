package planner

import (
	"testing"

	"github.com/Ltorre/palpedia-snapshot/internal/breeding"
)

func TestResolvePairAndTraitFilters(t *testing.T) {
	rules, err := breeding.Default()
	if err != nil {
		t.Fatal(err)
	}
	male := Pal{CharacterID: "SheepBall", Gender: "male", Traits: []string{"Test_PalEgg_HatchingSpeed_Up"}}
	female := Pal{CharacterID: "PinkCat", Gender: "female", Traits: []string{"PAL_ALLAttack_up3"}}
	result, err := ResolvePair(rules, male, female)
	if err != nil || result.Child == "" {
		t.Fatalf("pair result = %#v, err = %v", result, err)
	}
	if got := Filter([]Pal{male, female}, "", true, false); len(got) != 1 || got[0].CharacterID != "SheepBall" {
		t.Fatalf("gold filter = %#v", got)
	}
	if got := Filter([]Pal{male, female}, "demon god", false, true); len(got) != 1 || got[0].CharacterID != "PinkCat" {
		t.Fatalf("diamond filter = %#v", got)
	}
}

func TestShortestPathReturnsOwnedAndBreedingTargets(t *testing.T) {
	rules, err := breeding.Default()
	if err != nil {
		t.Fatal(err)
	}
	pals := []Pal{{CharacterID: "SheepBall", Gender: "male"}, {CharacterID: "PinkCat", Gender: "female"}}
	owned, err := ShortestPath(rules, pals, "SheepBall")
	if err != nil || owned.Generations != 0 || len(owned.Steps) != 0 {
		t.Fatalf("owned path = %#v, err = %v", owned, err)
	}
	target, err := ResolvePair(rules, pals[0], pals[1])
	if err != nil {
		t.Fatal(err)
	}
	path, err := ShortestPath(rules, pals, target.Child)
	if err != nil || path.Generations != 1 || len(path.Steps) != 1 {
		t.Fatalf("path = %#v, err = %v", path, err)
	}
}

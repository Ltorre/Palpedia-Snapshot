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

func TestFilterMatchesPlayerFacingName(t *testing.T) {
	pals := []Pal{{CharacterID: "BOSS_GrassMammoth", DisplayName: "Mammorest"}}
	if got := Filter(pals, "mammorest", false, false); len(got) != 1 {
		t.Fatalf("display-name filter = %#v", got)
	}
}

func TestFilterOptionsSexAndLevelSort(t *testing.T) {
	pals := []Pal{
		{InstanceID: "high", CharacterID: "SheepBall", DisplayName: "Lamball", Gender: "male", Level: 30},
		{InstanceID: "zero", CharacterID: "PinkCat", DisplayName: "Cattiva", Gender: "female", Level: 0},
		{InstanceID: "low", CharacterID: "ChickenPal", DisplayName: "Chikipi", Gender: "female", Level: 3},
	}
	females := FilterWithOptions(pals, FilterOptions{MaleOnly: false, FemaleOnly: true, SortOrder: SortByLevelAscending})
	if len(females) != 2 || females[0].InstanceID != "zero" || PalLevel(females[0]) != 1 || females[1].InstanceID != "low" {
		t.Fatalf("female level order = %#v", females)
	}
	opposite := FilterWithOptions(pals, FilterOptions{MaleOnly: true, RequiredGender: "female"})
	if len(opposite) != 2 || opposite[0].Gender != "female" || opposite[1].Gender != "female" {
		t.Fatalf("required gender should override manual filter: %#v", opposite)
	}
}

func TestFilterOptionsKeepsHighestEquivalentPal(t *testing.T) {
	pals := []Pal{
		{InstanceID: "lower", CharacterID: "SheepBall", Gender: "male", Level: 12, Traits: []string{"TraitA", "TraitB"}},
		{InstanceID: "higher", CharacterID: "SheepBall", Gender: "male", Level: 30, Traits: []string{"TraitB", "TraitA"}},
		{InstanceID: "different-sex", CharacterID: "SheepBall", Gender: "female", Level: 40, Traits: []string{"TraitA", "TraitB"}},
		{InstanceID: "different-trait", CharacterID: "SheepBall", Gender: "male", Level: 40, Traits: []string{"TraitA"}},
	}
	got := FilterWithOptions(pals, FilterOptions{Deduplicate: true})
	if len(got) != 3 {
		t.Fatalf("deduplicated Pals = %#v, want 3", got)
	}
	for _, pal := range got {
		if pal.InstanceID == "lower" {
			t.Fatalf("lower-level equivalent Pal was retained: %#v", got)
		}
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
	var inheritanceParents []Pal
	for _, male := range rules.Species() {
		for _, female := range rules.Species() {
			result, ok := rules.Resolve(male, "male", female, "female")
			if ok && result.Child == "SheepBall" && male != "SheepBall" && female != "SheepBall" {
				inheritanceParents = []Pal{{CharacterID: "SheepBall", Gender: "male"}, {CharacterID: male, Gender: "male"}, {CharacterID: female, Gender: "female"}}
				break
			}
		}
		if len(inheritanceParents) > 0 {
			break
		}
	}
	if len(inheritanceParents) == 0 {
		t.Fatal("no non-SheepBall parents found for inheritance-route test")
	}
	asIfUnowned, err := ShortestPathAsIfUnowned(rules, inheritanceParents, "SheepBall")
	if err != nil || asIfUnowned.Generations != 1 || len(asIfUnowned.Steps) != 1 {
		t.Fatalf("as-if-unowned path = %#v, err = %v", asIfUnowned, err)
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

func TestBreedingSpeedHelpers(t *testing.T) {
	helpers := BreedingSpeedHelpers([]Pal{
		{InstanceID: "ordinary", CharacterID: "SheepBall", Traits: []string{"CraftSpeed_up2"}},
		{InstanceID: "farm", CharacterID: "Chikipi", Traits: []string{"Test_PalEgg_HatchingSpeed_Up"}},
		{InstanceID: "base", CharacterID: "Lamball", Traits: []string{"MutationPal_Babysitter"}},
	})
	if len(helpers) != 2 {
		t.Fatalf("got %d helpers, want 2", len(helpers))
	}
	if helpers[0].TraitID != "MutationPal_Babysitter" || helpers[0].Pal.InstanceID != "base" {
		t.Fatalf("first helper = %#v, want Babysitter Pal", helpers[0])
	}
	if helpers[1].TraitID != "Test_PalEgg_HatchingSpeed_Up" || helpers[1].Pal.InstanceID != "farm" {
		t.Fatalf("second helper = %#v, want Philanthropist Pal", helpers[1])
	}
}

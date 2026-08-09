package breeding

import "testing"

func TestDefaultResolvesSpecialAndGenericRules(t *testing.T) {
	rules, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	special, ok := rules.Resolve("AmaterasuWolf", "male", "BadCatGirl", "female")
	if !ok || special.Child != "AmaterasuWolf_Dark" || special.Rule != "special" {
		t.Fatalf("special = %#v, ok=%v", special, ok)
	}
	generic, ok := rules.Resolve("Anubis", "male", "SheepBall", "female")
	if !ok || generic.Rule != "generic" || generic.TargetRank != 1765 || generic.Child == "" {
		t.Fatalf("generic = %#v, ok=%v", generic, ok)
	}
	same, ok := rules.Resolve("SheepBall", "male", "SheepBall", "female")
	if !ok || same.Child != "SheepBall" || same.Rule != "same_species" {
		t.Fatalf("same = %#v, ok=%v", same, ok)
	}
}

func TestDefaultSupportsGenderSpecificSpecialRules(t *testing.T) {
	rules, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	first, ok := rules.Resolve("CatMage", "male", "FoxMage", "female")
	if !ok || first.Child != "FoxMage_Dark" {
		t.Fatalf("first = %#v, ok=%v", first, ok)
	}
	second, ok := rules.Resolve("CatMage", "female", "FoxMage", "male")
	if !ok || second.Child != "CatMage_Fire" {
		t.Fatalf("second = %#v, ok=%v", second, ok)
	}
}

func TestDefaultCanonicalizesBossIDs(t *testing.T) {
	rules, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	result, ok := rules.Resolve("BOSS_SheepBall", "male", "SheepBall", "female")
	if !ok || result.Child != "SheepBall" {
		t.Fatalf("result = %#v, ok=%v", result, ok)
	}
}

package planner

import "testing"

func TestTraitCatalogTranslatesStandardPassiveTraits(t *testing.T) {
	tests := []struct {
		id, name string
	}{
		{"CraftSpeed_up1", "Serious"},
		{"PAL_ALLAttack_up1", "Brave"},
		{"MoveSpeed_up_1", "Nimble"},
		{"Test_PalEgg_HatchingSpeed_Up", "Philanthropist"},
	}
	for _, test := range tests {
		if got := TraitName(test.id); got != test.name {
			t.Errorf("TraitName(%q) = %q, want %q", test.id, got, test.name)
		}
		if got := TraitEffect(test.id); got == "" {
			t.Errorf("TraitEffect(%q) is empty", test.id)
		}
	}
}

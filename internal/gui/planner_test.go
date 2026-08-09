//go:build windows

package gui

import (
	"testing"

	"github.com/Ltorre/palpedia-snapshot/internal/planner"
)

func TestPlannerRequiredGender(t *testing.T) {
	s := &screen{}
	male := planner.Pal{Gender: "male"}
	female := planner.Pal{Gender: "female"}
	s.selectedMale = &male
	if got := s.plannerRequiredGender(); got != "female" {
		t.Fatalf("after male selection required gender = %q, want female", got)
	}
	s.selectedMale, s.selectedFemale = nil, &female
	if got := s.plannerRequiredGender(); got != "male" {
		t.Fatalf("after female selection required gender = %q, want male", got)
	}
	s.selectedMale = &male
	if got := s.plannerRequiredGender(); got != "" {
		t.Fatalf("after both selections required gender = %q, want empty", got)
	}
}

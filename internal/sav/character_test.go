package sav

import "testing"

func TestPalLevelMinimum(t *testing.T) {
	if got := palLevel(0); got != 1 {
		t.Fatalf("palLevel(0) = %d, want 1", got)
	}
	if got := palLevel(12); got != 12 {
		t.Fatalf("palLevel(12) = %d, want 12", got)
	}
}

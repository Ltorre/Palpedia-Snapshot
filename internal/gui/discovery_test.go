package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindSavesFindsWorldDirectories(t *testing.T) {
	root := t.TempDir()
	world := filepath.Join(root, "steam-id", "world-id")
	if err := os.MkdirAll(world, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(world, "Level.sav"), []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	candidates, err := FindSaves(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].WorldDir != world || candidates[0].SteamID != "steam-id" {
		t.Fatalf("candidates = %#v", candidates)
	}
}

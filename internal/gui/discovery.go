package gui

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

type SaveCandidate struct {
	LevelPath string
	WorldDir  string
	SteamID   string
	UpdatedAt time.Time
}

func DefaultSaveRoot() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "AppData", "Local", "Pal", "Saved", "SaveGames")
}

func FindSaves(root string) ([]SaveCandidate, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	add := func(dir, steamID string, candidates *[]SaveCandidate) {
		level := filepath.Join(dir, "Level.sav")
		info, err := os.Stat(level)
		if err != nil || !info.Mode().IsRegular() {
			return
		}
		if _, exists := seen[level]; exists {
			return
		}
		seen[level] = struct{}{}
		*candidates = append(*candidates, SaveCandidate{LevelPath: level, WorldDir: dir, SteamID: steamID, UpdatedAt: info.ModTime()})
	}

	candidates := make([]SaveCandidate, 0)
	add(root, "", &candidates)
	steamDirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, steamDir := range steamDirs {
		if !steamDir.IsDir() {
			continue
		}
		steamPath := filepath.Join(root, steamDir.Name())
		add(steamPath, steamDir.Name(), &candidates)
		worldDirs, readErr := os.ReadDir(steamPath)
		if readErr != nil {
			continue
		}
		for _, worldDir := range worldDirs {
			if worldDir.IsDir() {
				add(filepath.Join(steamPath, worldDir.Name()), steamDir.Name(), &candidates)
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].UpdatedAt.After(candidates[j].UpdatedAt) })
	return candidates, nil
}

func DefaultExportDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Documents", "Palpedia Snapshot Exports")
}

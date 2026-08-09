//go:build windows

package sav

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

var steamLibraryPath = regexp.MustCompile(`(?m)"path"\s+"([^"]+)"`)

func findInstalledOodleLibrary() string {
	for _, library := range steamLibraries() {
		candidate := filepath.Join(library, "steamapps", "common", "Palworld", "Pal", "Binaries", "Win64", oodleLibrary)
		if isRegularFile(candidate) {
			return candidate
		}
	}
	return ""
}

func steamLibraries() []string {
	libraries := make([]string, 0, 4)
	add := func(path string) {
		if path == "" {
			return
		}
		path = filepath.Clean(path)
		for _, library := range libraries {
			if strings.EqualFold(library, path) {
				return
			}
		}
		libraries = append(libraries, path)
	}
	for _, path := range steamInstallPaths() {
		add(path)
		data, err := os.ReadFile(filepath.Join(path, "steamapps", "libraryfolders.vdf"))
		if err != nil {
			continue
		}
		for _, match := range steamLibraryPath.FindAllStringSubmatch(string(data), -1) {
			add(strings.ReplaceAll(match[1], `\\`, `\`))
		}
	}
	return libraries
}

func steamInstallPaths() []string {
	paths := []string{}
	if programFiles := os.Getenv("ProgramFiles(x86)"); programFiles != "" {
		paths = append(paths, filepath.Join(programFiles, "Steam"))
	}
	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		paths = append(paths, filepath.Join(programFiles, "Steam"))
	}
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return paths
	}
	defer key.Close()
	if installPath, _, err := key.GetStringValue("SteamPath"); err == nil {
		paths = append(paths, installPath)
	}
	return paths
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

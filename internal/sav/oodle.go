package sav

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	oodle "github.com/new-world-tools/go-oodle"
)

var oodleSetup struct {
	sync.Mutex
	ready bool
}

const oodleLibrary = "oo2core_9_win64.dll"

func oodleDecompress(src []byte, rawLen int) ([]byte, error) {
	oodleSetup.Lock()
	if !oodleSetup.ready {
		if err := prepareOodle(); err != nil {
			oodleSetup.Unlock()
			return nil, err
		}
		oodleSetup.ready = true
	}
	oodleSetup.Unlock()
	out, err := oodle.Decompress(src, int64(rawLen))
	if err != nil {
		return nil, fmt.Errorf("sav: Oodle decompress: %w", err)
	}
	return out, nil
}

func prepareOodle() error {
	path, err := resolveOodleLibrary()
	if err != nil {
		return err
	}
	if err := oodle.LoadFrom(path); err != nil {
		return fmt.Errorf("sav: load Oodle library: %w", err)
	}
	return nil
}

func resolveOodleLibrary() (string, error) {
	path := os.Getenv("PALWORLD_SCRAP_OODLE_LIB")
	if path == "" {
		path = findInstalledOodleLibrary()
	}
	if path == "" {
		return "", fmt.Errorf("sav: Oodle DLL was not found automatically; choose oo2core_9_win64.dll in the optional advanced options")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("sav: PALWORLD_SCRAP_OODLE_LIB must be an absolute path: %q", path)
	}
	st, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("sav: PALWORLD_SCRAP_OODLE_LIB: %w", err)
	}
	if !st.Mode().IsRegular() {
		return "", fmt.Errorf("sav: PALWORLD_SCRAP_OODLE_LIB is not a regular file: %s", path)
	}
	return path, nil
}

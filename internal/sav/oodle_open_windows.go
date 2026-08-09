//go:build windows

package sav

import (
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed assets/ooz_windows_amd64.exe
var oozWindowsBinary []byte

func oodleDecompress(src []byte, rawLen int) ([]byte, error) {
	if rawLen <= 0 {
		return nil, fmt.Errorf("sav: invalid Oodle output length %d", rawLen)
	}
	helper, err := oozHelperPath()
	if err != nil {
		return nil, err
	}
	workDir, err := os.MkdirTemp("", "palworld-save-scrap-oodle-")
	if err != nil {
		return nil, fmt.Errorf("sav: create Oodle work directory: %w", err)
	}
	defer os.RemoveAll(workDir)
	inputPath := filepath.Join(workDir, "input.ooz")
	outputPath := filepath.Join(workDir, "output.gvas")
	input := make([]byte, 8+len(src))
	binary.LittleEndian.PutUint64(input, uint64(rawLen))
	copy(input[8:], src)
	if err := os.WriteFile(inputPath, input, 0o600); err != nil {
		return nil, fmt.Errorf("sav: write Oodle input: %w", err)
	}
	output, err := exec.Command(helper, inputPath, outputPath).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sav: built-in Oodle decoder failed: %w: %s", err, output)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("sav: read Oodle output: %w", err)
	}
	return raw, nil
}

func oozHelperPath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("sav: find Oodle cache directory: %w", err)
	}
	digest := sha256.Sum256(oozWindowsBinary)
	path := filepath.Join(cacheDir, "Palworld Save Scrap", fmt.Sprintf("ooz-%x.exe", digest[:8]))
	if data, err := os.ReadFile(path); err == nil && sha256.Sum256(data) == digest {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("sav: create Oodle cache directory: %w", err)
	}
	if err := os.WriteFile(path, oozWindowsBinary, 0o700); err != nil {
		return "", fmt.Errorf("sav: write built-in Oodle decoder: %w", err)
	}
	return path, nil
}

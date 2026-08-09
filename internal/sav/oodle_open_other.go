//go:build !windows

package sav

import "fmt"

func oodleDecompress(_ []byte, _ int) ([]byte, error) {
	return nil, fmt.Errorf("sav: Oodle-compressed saves require the Windows build")
}

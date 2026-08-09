//go:build windows

package oodle

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

var library struct {
	sync.Once
	err        error
	decompress *syscall.Proc
}

func LoadFrom(absPath string) error {
	library.Do(func() {
		if !filepath.IsAbs(absPath) {
			library.err = fmt.Errorf("oodle: library path must be absolute: %q", absPath)
			return
		}
		st, err := os.Stat(absPath)
		if err != nil {
			library.err = fmt.Errorf("oodle: stat %s: %w", absPath, err)
			return
		}
		if !st.Mode().IsRegular() {
			library.err = fmt.Errorf("oodle: library path is not a regular file: %s", absPath)
			return
		}
		dll, err := syscall.LoadDLL(absPath)
		if err != nil {
			library.err = fmt.Errorf("oodle: load %s: %w", absPath, err)
			return
		}
		library.decompress, library.err = dll.FindProc("OodleLZ_Decompress")
	})
	return library.err
}

func Decompress(input []byte, outputSize int64) ([]byte, error) {
	if len(input) == 0 || outputSize <= 0 {
		return nil, fmt.Errorf("oodle: invalid input/output size")
	}
	if library.err != nil {
		return nil, library.err
	}
	if library.decompress == nil {
		return nil, fmt.Errorf("oodle: library has not been loaded")
	}
	out := make([]byte, outputSize)
	n, _, _ := library.decompress.Call(
		uintptr(unsafe.Pointer(&input[0])), uintptr(len(input)),
		uintptr(unsafe.Pointer(&out[0])), uintptr(outputSize),
		0, 0, 0, 0, 0, 0, 0, 0, 0, 3,
	)
	if n == 0 {
		return nil, fmt.Errorf("oodle: decompression failed")
	}
	if int64(n) != outputSize {
		return nil, fmt.Errorf("oodle: decompressed %d bytes, expected %d", n, outputSize)
	}
	return out, nil
}

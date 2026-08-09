//go:build windows

package gui

import (
	"errors"
	"os/exec"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	bifReturnOnlyFSDirs = 0x0001
	bifEditBox          = 0x0010
	bifNewDialogStyle   = 0x0040
)

var (
	shell32                   = syscall.NewLazyDLL("shell32.dll")
	ole32                     = syscall.NewLazyDLL("ole32.dll")
	shBrowseForFolder         = shell32.NewProc("SHBrowseForFolderW")
	shGetPathFromIDList       = shell32.NewProc("SHGetPathFromIDListW")
	coTaskMemFree             = ole32.NewProc("CoTaskMemFree")
	errFolderSelectionAborted = errors.New("folder selection was cancelled")
)

type browseInfo struct {
	owner       uintptr
	root        uintptr
	displayName *uint16
	title       *uint16
	flags       uint32
	callback    uintptr
	lParam      uintptr
	image       int32
}

func chooseFolder(title string) (string, error) {
	titleUTF16, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return "", err
	}
	pathUTF16 := make([]uint16, windows.MAX_PATH)
	info := browseInfo{
		displayName: &pathUTF16[0],
		title:       titleUTF16,
		flags:       bifReturnOnlyFSDirs | bifEditBox | bifNewDialogStyle,
	}
	pidl, _, _ := shBrowseForFolder.Call(uintptr(unsafe.Pointer(&info)))
	if pidl == 0 {
		return "", errFolderSelectionAborted
	}
	defer coTaskMemFree.Call(pidl)
	ok, _, _ := shGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&pathUTF16[0])))
	if ok == 0 {
		return "", errors.New("selected folder path is unavailable")
	}
	return windows.UTF16ToString(pathUTF16), nil
}

func openFolder(path string) error {
	return exec.Command("explorer.exe", path).Start()
}

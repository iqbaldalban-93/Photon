//go:build windows

package hostdelete

import (
	"fmt"
	"path/filepath"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	foDelete           = 3
	fofSilent          = 0x4
	fofNoConfirmation  = 0x10
	fofAllowUndo       = 0x40
	fofNoErrorUI       = 0x400
	fofWantNukeWarning = 0x4000
)

var procSHFileOperationW = windows.NewLazySystemDLL("shell32.dll").NewProc("SHFileOperationW")

type shFileopstructW struct {
	hwnd              windows.HWND
	wFunc             uint32
	_                 uint32
	pFrom             *uint16
	pTo               *uint16
	fFlags            uint16
	_                 uint16
	fAnyOpsAborted    int32
	hNameMappings     uintptr
	lpszProgressTitle *uint16
}

func Trash(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	from, err := windows.UTF16FromString(abs)
	if err != nil {
		return err
	}
	from = append(from, 0)

	op := shFileopstructW{
		wFunc:  foDelete,
		pFrom:  &from[0],
		fFlags: fofAllowUndo | fofNoConfirmation | fofSilent | fofNoErrorUI | fofWantNukeWarning,
	}
	ret, _, _ := procSHFileOperationW.Call(uintptr(unsafe.Pointer(&op)))
	if ret != 0 {
		return fmt.Errorf("recycle bin delete failed (code %d): %s", ret, abs)
	}
	if op.fAnyOpsAborted != 0 {
		return fmt.Errorf("recycle bin delete was cancelled; file kept: %s", abs)
	}
	return nil
}

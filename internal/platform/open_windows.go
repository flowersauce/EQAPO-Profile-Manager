//go:build windows

package platform

import (
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var shOpenWithDialog = windows.NewLazySystemDLL("shell32.dll").NewProc("SHOpenWithDialog")

type openAsInfo struct {
	File, Class *uint16
	Flags       uint32
}

// OpenWith asks Windows to choose an application for one existing file.
// It runs in the caller's normal process, never in the elevated worker.
func OpenWith(path string) error {
	if _, err := Regular(path); err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err != nil && err != syscall.Errno(1) {
		return err
	}
	defer windows.CoUninitialize()
	info := openAsInfo{File: file, Flags: 0x00000004} // OAIF_EXEC: open with the chosen app.
	result, _, _ := shOpenWithDialog.Call(0, uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(info)
	return openWithResult(uint32(result))
}

func openWithResult(result uint32) error {
	if result == 0x800704C7 || result == 1 {
		return nil
	} // Cancelled / S_FALSE.
	if int32(result) < 0 {
		return syscall.Errno(result)
	}
	return nil
}

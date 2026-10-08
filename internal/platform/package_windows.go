//go:build windows

package platform

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var getCurrentPackageFamilyName = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetCurrentPackageFamilyName")

// PackageFamilyName returns an empty name for the MSI and portable executable.
func PackageFamilyName() (string, error) {
	var length uint32
	code, _, _ := getCurrentPackageFamilyName.Call(uintptr(unsafe.Pointer(&length)), 0)
	const noPackage = 15700 // APPMODEL_ERROR_NO_PACKAGE
	if code == noPackage {
		return "", nil
	}
	if code != uintptr(windows.ERROR_INSUFFICIENT_BUFFER) {
		return "", syscall.Errno(code)
	}
	buffer := make([]uint16, length)
	code, _, _ = getCurrentPackageFamilyName.Call(uintptr(unsafe.Pointer(&length)), uintptr(unsafe.Pointer(&buffer[0])))
	if code != 0 {
		return "", syscall.Errno(code)
	}
	return windows.UTF16ToString(buffer), nil
}

//go:build windows

package platform

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"golang.org/x/sys/windows"
)

const elevationArgument = "--eqm-elevated-context"

// ElevationContext keeps over-the-shoulder UAC in the invoking user's profile.
// Operations travel only over the authenticated, process-bound local pipe.
type ElevationContext struct {
	Pipe         string `json:"pipe"`
	Parent       uint32 `json:"parent"`
	LocalAppData string `json:"local_app_data"`
	Locale       string `json:"locale"`
}

func LocalAppData() (string, error) { return windows.KnownFolderPath(windows.FOLDERID_LocalAppData, 0) }
func Elevated() bool                { return windows.GetCurrentProcessToken().IsElevated() }

func PermissionDenied(err error) bool {
	return errors.Is(err, os.ErrPermission) || errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD)
}

// DecodeElevation strips only the private UAC transport, leaving CLI args intact.
func DecodeElevation(args []string, elevated bool) (ElevationContext, []string, bool, error) {
	var context ElevationContext
	if len(args) == 0 || args[0] != elevationArgument {
		return context, args, false, nil
	}
	if !elevated || len(args) < 2 {
		return context, nil, false, fault.New("elevationContext")
	}
	data, err := base64.RawURLEncoding.DecodeString(args[1])
	if err != nil || len(data) > 64*1024 {
		return context, nil, false, fault.New("elevationContext")
	}
	if err := json.Unmarshal(data, &context); err != nil || !filepath.IsAbs(context.LocalAppData) || context.Locale == "" || !strings.HasPrefix(context.Pipe, `\\.\pipe\eqm-`) || context.Parent == 0 {
		return context, nil, false, fault.New("elevationContext")
	}
	return context, args[2:], true, nil
}

func elevationArgs(args []string, context ElevationContext) ([]string, error) {
	data, err := json.Marshal(context)
	if err != nil {
		return nil, err
	}
	result := []string{elevationArgument, base64.RawURLEncoding.EncodeToString(data)}
	return append(result, args...), nil
}

// shellExecuteInfo follows SHELLEXECUTEINFOW, including its handle union.
type shellExecuteInfo struct {
	Size, Mask                        uint32
	Window                            windows.Handle
	Verb, File, Parameters, Directory *uint16
	Show                              int32
	Instance                          windows.Handle
	IDList                            uintptr
	Class                             *uint16
	ClassKey                          windows.Handle
	HotKey                            uint32
	Icon                              windows.Handle
	Process                           windows.Handle
}

var shellExecuteEx = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

// startElevated starts a hidden worker. The original process retains all UI.
func startElevated(args []string, context ElevationContext) (windows.Handle, error) {
	if Elevated() {
		return 0, fault.New("elevationDenied")
	}
	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return 0, err
	}
	forwarded, err := elevationArgs(args, context)
	if err != nil {
		return 0, err
	}
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return 0, err
	}
	params, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(forwarded))
	if err != nil {
		return 0, err
	}
	dir, err := windows.UTF16PtrFromString(cwd)
	if err != nil {
		return 0, err
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// S_FALSE (1) is also a successful COM initialization and needs balancing.
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED|windows.COINIT_DISABLE_OLE1DDE); err != nil && err != syscall.Errno(1) {
		return 0, fault.Wrap("elevationFailed", err)
	}
	defer windows.CoUninitialize()
	info := shellExecuteInfo{
		Mask: 0x00000040 | 0x00000100 | 0x00000400, // NOCLOSEPROCESS | NOASYNC | FLAG_NO_UI
		Verb: verb, File: file, Parameters: params, Directory: dir, Show: windows.SW_HIDE,
	}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, callErr := shellExecuteEx.Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(info)
	if ok == 0 {
		if errors.Is(callErr, windows.ERROR_CANCELLED) {
			return 0, fault.New("elevationCancelled")
		}
		return 0, fault.Wrap("elevationFailed", callErr)
	}
	if info.Process == 0 {
		return 0, fault.New("elevationFailed")
	}
	return info.Process, nil
}

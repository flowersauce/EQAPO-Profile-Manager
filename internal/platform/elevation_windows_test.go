package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestElevationTransportPreservesArgumentsAndUserContext(t *testing.T) {
	context := ElevationContext{LocalAppData: filepath.Join(t.TempDir(), "原用户 空格"), Locale: "zh-CN", Pipe: `\\.\pipe\eqm-test`, Parent: uint32(os.Getpid())}
	original := []string{"import", "space and 中文", `quote"inside`, "", `C:\trailing\`}
	forwarded, err := elevationArgs(original, context)
	if err != nil {
		t.Fatal(err)
	}
	forwarded, err = windows.DecomposeCommandLine(windows.ComposeCommandLine(forwarded))
	if err != nil {
		t.Fatal(err)
	}
	gotContext, gotArgs, child, err := DecodeElevation(forwarded, true)
	if err != nil || !child || gotContext != context || !reflect.DeepEqual(gotArgs, original) {
		t.Fatalf("transport changed: %+v %q %v", gotContext, gotArgs, err)
	}
	if _, _, _, err := DecodeElevation(forwarded, false); err == nil {
		t.Fatal("unelevated process accepted private transport")
	}
	plainContext, plainArgs, child, err := DecodeElevation([]string{"list"}, false)
	if err != nil || child || plainContext != (ElevationContext{}) || !reflect.DeepEqual(plainArgs, []string{"list"}) {
		t.Fatal("ordinary arguments were changed")
	}
}

func TestOnlyPermissionErrorsRequestElevation(t *testing.T) {
	for _, err := range []error{os.ErrPermission, windows.ERROR_ACCESS_DENIED, windows.ERROR_PRIVILEGE_NOT_HELD, fmt.Errorf("wrapped: %w", windows.ERROR_ACCESS_DENIED)} {
		if !PermissionDenied(err) {
			t.Errorf("missed permission error: %v", err)
		}
	}
	for _, err := range []error{nil, os.ErrNotExist, windows.ERROR_SHARING_VIOLATION, windows.ERROR_CANCELLED} {
		if PermissionDenied(err) {
			t.Errorf("unexpected elevation: %v", err)
		}
	}
}

func TestShellExecuteInfoNativeLayout(t *testing.T) {
	var info shellExecuteInfo
	if unsafe.Sizeof(uintptr(0)) == 8 {
		if unsafe.Sizeof(info) != 112 || unsafe.Offsetof(info.Process) != 104 {
			t.Fatal("incorrect 64-bit SHELLEXECUTEINFOW layout")
		}
	} else if unsafe.Sizeof(info) != 60 || unsafe.Offsetof(info.Process) != 56 {
		t.Fatal("incorrect 32-bit SHELLEXECUTEINFOW layout")
	}
}

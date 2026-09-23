//go:build windows

// Package platform contains the Windows-specific filesystem and console boundary.
package platform

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"golang.org/x/sys/windows"
)

// Username returns the account name rather than its optional display name.
func Username() string {
	account, err := user.Current()
	if err != nil {
		return ""
	}
	name := account.Username
	return name[strings.LastIndex(name, `\`)+1:]
}

// UILanguage returns the first Windows user display language, not the region format.
func UILanguage() string {
	languages, err := windows.GetUserPreferredUILanguages(windows.MUI_LANGUAGE_NAME)
	if err == nil && len(languages) > 0 {
		return languages[0]
	}
	return "en"
}

func IsTerminal(f *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(f.Fd()), &mode) == nil
}

// UTF8Console restores the shell's code pages when the command returns.
func UTF8Console() (func(), error) {
	restoreInput, restoreOutput := func() {}, func() {}
	restore := func() { restoreInput(); restoreOutput() }
	if IsTerminal(os.Stdin) {
		cp, err := windows.GetConsoleCP()
		if err != nil {
			return restore, err
		}
		if err := windows.SetConsoleCP(65001); err != nil {
			return restore, err
		}
		restoreInput = func() { _ = windows.SetConsoleCP(cp) }
	}
	if IsTerminal(os.Stdout) || IsTerminal(os.Stderr) {
		cp, err := windows.GetConsoleOutputCP()
		if err != nil {
			return restore, err
		}
		if err := windows.SetConsoleOutputCP(65001); err != nil {
			return restore, err
		}
		restoreOutput = func() { _ = windows.SetConsoleOutputCP(cp) }
	}
	return restore, nil
}

// Regular rejects reparse points before callers read or mutate managed files.
func Regular(path string) (os.FileInfo, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	attrs, err := windows.GetFileAttributes(ptr)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return nil, fault.New("regular", path)
	}
	return info, nil
}

func Directory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attrs, err := windows.GetFileAttributes(ptr)
	if err != nil {
		return err
	}
	if !info.IsDir() || attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		return fault.New("directory", path)
	}
	return nil
}

// WithLock serializes eqm commits without holding a lock during user input.
func WithLock(dir string, action func() error) error {
	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(strings.ToUpper(filepath.Clean(canonical))))
	name, err := windows.UTF16PtrFromString(fmt.Sprintf(`Global\eqm-%x`, digest))
	if err != nil {
		return err
	}
	// Windows mutex ownership belongs to an OS thread, not a Go goroutine.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h, err := windows.CreateMutex(nil, false, name)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return err
	}
	defer windows.CloseHandle(h)
	status, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		return err
	}
	if status != windows.WAIT_OBJECT_0 && status != windows.WAIT_ABANDONED {
		return fault.New("busy")
	}
	defer windows.ReleaseMutex(h)
	return action()
}

func replaceFile(source, target string, exists bool) error {
	src, err := windows.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	flags := uint32(windows.MOVEFILE_WRITE_THROUGH)
	if exists {
		flags |= windows.MOVEFILE_REPLACE_EXISTING
	}
	return windows.MoveFileEx(src, dst, flags)
}

// Rename never replaces another file that appeared after validation.
func Rename(source, target string) error { return replaceFile(source, target, false) }

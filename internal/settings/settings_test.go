package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortableFlagSelectsSettingsWithoutCreatingFiles(t *testing.T) {
	exeDir := t.TempDir()
	exe := filepath.Join(exeDir, "eqm.exe")
	local := t.TempDir()
	installed := filepath.Join(local, "EQM", "config.json")
	got, err := PathFor(exe, local)
	if err != nil || got != installed {
		t.Fatalf("installed: %q %v", got, err)
	}
	if _, err := os.Stat(filepath.Dir(installed)); !os.IsNotExist(err) {
		t.Fatal("path resolution wrote a directory")
	}
	flag := filepath.Join(exeDir, "portable.flag")
	if err := os.WriteFile(flag, nil, 0644); err != nil {
		t.Fatal(err)
	}
	got, err = PathFor(exe, local)
	if err != nil || got != filepath.Join(exeDir, "config", "config.json") {
		t.Fatalf("portable: %q %v", got, err)
	}
	if err := os.Remove(flag); err != nil {
		t.Fatal(err)
	}
	got, err = PathFor(exe, local)
	if err != nil || got != installed {
		t.Fatalf("removed flag: %q %v", got, err)
	}
}

func TestSettingsLocationDoesNotFallbackOnInvalidMarker(t *testing.T) {
	exeDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(exeDir, "portable.flag"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := PathFor(filepath.Join(exeDir, "eqm.exe"), t.TempDir()); err == nil {
		t.Fatal("invalid marker silently fell back to installed mode")
	}
	if _, err := PathFor("eqm.exe", t.TempDir()); err == nil {
		t.Fatal("accepted working-directory-relative exe")
	}
}

func TestInstalledLocationUsesOriginalUserAfterUAC(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "eqm.exe")
	original := t.TempDir()
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "OtherAdmin"))
	got, err := PathFor(exe, original)
	if err != nil || got != filepath.Join(original, "EQM", "config.json") {
		t.Fatalf("changed user context: %q %v", got, err)
	}
}

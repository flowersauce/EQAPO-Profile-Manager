package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/apo"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
)

func TestFlatInventoryAndProfileBoundaries(t *testing.T) {
	s := fixture(t)
	data := []byte("\xef\xbb\xbfGraphicEQ: 20 -3; 1000 0\r\n")
	for _, name := range []string{"无后缀", "custom.cfg", "A.txt"} {
		plan := importFile(t, s, name, data)
		got, err := os.ReadFile(plan.Target.Path)
		if err != nil || plan.Filename != name || !bytes.Equal(got, data) {
			t.Fatalf("flat import changed name or bytes: %q %v", name, err)
		}
	}
	nested := s.ProfilePath(filepath.Join("旧分类", "hidden.txt"))
	if err := os.Mkdir(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}
	write(t, nested, data)
	write(t, s.ProfilePath("notes.md"), []byte("unrelated text"))
	state := readState(t, s)
	if len(state.Profiles) != 3 || state.Profiles[0].Filename != "A.txt" {
		t.Fatalf("inventory is not a sorted root-only list: %+v", state.Profiles)
	}
	name := filepath.Join("旧分类", "hidden.txt")
	if _, err := s.ReadProfile(name); err == nil {
		t.Fatal("read accepted a nested profile")
	}
	if err := s.Switch(state, name); err == nil {
		t.Fatal("switch accepted a nested profile")
	}
	source, err := platform.Read(nested, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Rename(state, source, "renamed"); err == nil {
		t.Fatal("rename accepted a nested profile")
	}
	if err := s.Remove(state, source); err == nil {
		t.Fatal("remove accepted a nested profile")
	}
	if err := s.Import(state, ImportPlan{Source: source, Target: source, Filename: name}); err == nil {
		t.Fatal("import accepted a nested destination")
	}
	if err := source.Check(); err != nil {
		t.Fatal("rejected nested operations changed the file")
	}
}

func TestLegacyCategoryCanBeReimportedWithoutChangingItsSource(t *testing.T) {
	s := fixture(t)
	name := filepath.Join("旧分类", "耳机.txt")
	sourcePath := s.ProfilePath(name)
	if err := os.Mkdir(filepath.Dir(sourcePath), 0755); err != nil {
		t.Fatal(err)
	}
	data := []byte("GraphicEQ: 20 0")
	write(t, sourcePath, data)
	legacy := []byte(apo.Begin + "\nInclude: eqm-profiles\\" + name + "\n" + apo.End + "\n")
	write(t, filepath.Join(s.ConfigDir(), "config.txt"), legacy)
	state := readState(t, s)
	if state.Current != "" || len(state.Profiles) != 0 || !bytes.Equal(state.Config.Data, legacy) {
		t.Fatal("legacy read changed config or exposed a nested profile")
	}
	plan, err := s.PrepareImport(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Import(state, plan); err != nil {
		t.Fatal(err)
	}
	state = readState(t, s)
	if state.Document.Enabled || plan.Filename != "耳机.txt" || len(state.Profiles) != 1 {
		t.Fatal("reimport did not restore a flat profile and disable the legacy reference")
	}
	for _, path := range []string{sourcePath, s.ProfilePath("耳机.txt")} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatalf("reimport changed or lost a file: %s %v", path, err)
		}
	}
	if err := s.Switch(state, "耳机.txt"); err != nil {
		t.Fatal(err)
	}
	if readState(t, s).Current != "耳机.txt" {
		t.Fatal("reimported root profile cannot be applied")
	}
}

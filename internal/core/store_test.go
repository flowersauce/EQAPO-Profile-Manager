package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
)

func fixture(t *testing.T) Store {
	t.Helper()
	root := filepath.Join(t.TempDir(), "Equalizer APO 中文")
	if err := os.MkdirAll(filepath.Join(root, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "config", "config.txt"), []byte("Preamp: -1 dB\r\n"))
	plan, err := PrepareInit(root, filepath.Join(t.TempDir(), "config", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Commit(); err != nil {
		t.Fatal(err)
	}
	return plan.Store
}

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func readState(t *testing.T, s Store) State {
	t.Helper()
	state, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func importFile(t *testing.T, s Store, name string, data []byte) ImportPlan {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	write(t, path, data)
	p, err := s.PrepareImport(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Import(readState(t, s), p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestProfileLifecycleAndReadOnlyHealing(t *testing.T) {
	s := fixture(t)
	data := []byte("\xef\xbb\xbf# AutoEQ\r\nGraphicEQ: 20 -3; 1000 0\r\n")
	p := importFile(t, s, "耳机.txt", data)
	got, err := os.ReadFile(p.Target.Path)
	if err != nil || !bytes.Equal(data, got) {
		t.Fatal("import did not preserve bytes")
	}
	state := readState(t, s)
	if state.Current != "" {
		t.Fatal("import applied automatically")
	}
	if err := s.Switch(state, p.Filename); err != nil {
		t.Fatal(err)
	}
	state = readState(t, s)
	source, err := s.ReadProfile(p.Filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Rename(state, source, "新耳机"); err != nil {
		t.Fatal(err)
	}
	state = readState(t, s)
	if state.Current != "新耳机.txt" {
		t.Fatal("rename did not update Include")
	}
	before := bytes.Clone(state.Config.Data)
	if err := os.Remove(s.ProfilePath(state.Current)); err != nil {
		t.Fatal(err)
	}
	state = readState(t, s)
	if state.Current != "" || !bytes.Equal(before, state.Config.Data) {
		t.Fatal("read either kept a missing selection or wrote a repair")
	}
	if err := s.Switch(state, ""); err != nil {
		t.Fatal(err)
	}
	if readState(t, s).Document.Enabled {
		t.Fatal("commit failed to heal")
	}
}

func TestOverwriteIdentityAndSourceCleanup(t *testing.T) {
	s := fixture(t)
	p := importFile(t, s, "First.TXT", []byte("GraphicEQ: 20 0; 1000 -2"))
	if _, err := s.PrepareImport(p.Target.Path); !fault.IsKey(err, "sameFile") {
		t.Fatalf("same source accepted: %v", err)
	}
	alias := filepath.Join(t.TempDir(), "First.TXT")
	if err := os.Link(p.Target.Path, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := s.PrepareImport(alias); !fault.IsKey(err, "sameFile") {
		t.Fatalf("hardlink accepted: %v", err)
	}
	path := filepath.Join(t.TempDir(), "first.txt")
	write(t, path, []byte("GraphicEQ: 20 -4; 1000 -1"))
	next, err := s.PrepareImport(path)
	if err != nil || next.Filename != "First.TXT" || next.Target.Info == nil {
		t.Fatalf("overwrite plan: %+v %v", next, err)
	}
	if err := s.Import(readState(t, s), next); err != nil {
		t.Fatal(err)
	}
	write(t, path, []byte("changed after import"))
	if err := next.DeleteSource(); !fault.IsKey(err, "sourceKept") {
		t.Fatalf("changed source removed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("source was deleted")
	}
}

func TestConflictDoesNotOverwriteExternalConfig(t *testing.T) {
	s := fixture(t)
	state := readState(t, s)
	external := append(bytes.Clone(state.Config.Data), []byte("\n# external edit\n")...)
	write(t, state.Config.Path, external)
	if err := s.Switch(state, ""); !fault.IsKey(err, "changed") {
		t.Fatalf("expected conflict, got %v", err)
	}
	got, _ := os.ReadFile(state.Config.Path)
	if !bytes.Equal(got, external) {
		t.Fatal("external edit lost")
	}
}

func TestRemoveActiveAndRenameCase(t *testing.T) {
	s := fixture(t)
	p := importFile(t, s, "Sample.txt", []byte("Preamp: -3 dB\nFilter 1: ON PK Fc 100 Hz Gain -2 dB Q 1.0\n"))
	if err := s.Switch(readState(t, s), p.Filename); err != nil {
		t.Fatal(err)
	}
	source, _ := s.ReadProfile(p.Filename)
	if err := s.Rename(readState(t, s), source, "SAMPLE"); err != nil {
		t.Fatal(err)
	}
	state := readState(t, s)
	if state.Current != "SAMPLE.txt" {
		t.Fatalf("case change not reflected: %s", state.Current)
	}
	source, _ = s.ReadProfile(state.Current)
	if err := s.Remove(state, source); err != nil {
		t.Fatal(err)
	}
	state = readState(t, s)
	if state.Current != "" || len(state.Profiles) != 0 || state.Document.Enabled {
		t.Fatal("remove left active selection")
	}
}

func TestReadOnlyTargetFailureKeepsOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.txt")
	write(t, path, []byte("original"))
	snapshot, err := platform.Read(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })
	if err := platform.Write(snapshot, []byte("replacement")); err == nil {
		t.Fatal("expected read-only replacement failure")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "original" {
		t.Fatal("original lost on failed write")
	}
}

func TestFormatAndPathValidation(t *testing.T) {
	for _, text := range []string{"# GraphicEQ: 20 0", "Preamp: -1 dB", "GraphicEQ: NaN 0", "GraphicEQ: 20 Inf", "Include: other.txt", "Filter 1: ON PK Fc -1 Hz Gain 0 dB Q 1"} {
		if CheckFormat([]byte(text)) == nil {
			t.Errorf("accepted %q", text)
		}
	}
	for _, path := range []string{`"C:\one.txt" "C:\two.txt"`, "C:\\one.txt\nC:\\two.txt", ""} {
		if _, err := ParsePath(path); err == nil {
			t.Errorf("accepted multiple/empty paths: %q", path)
		}
	}
	got, err := ParsePath(`"C:\中文 路径\example.txt"`)
	if err != nil || got != `C:\中文 路径\example.txt` {
		t.Fatalf("path lost: %q %v", got, err)
	}
}

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathSuggestionMatchesTabWithoutConsumingCycle(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Headphones A.txt", "Headphones B.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	input := filepath.Join(root, "Head")
	p := newPathCompleter(false)
	first := p.Suggest(input)
	if first == "" || !strings.HasPrefix(first, input) || p.Suggest(input) != first {
		t.Fatal("unstable or missing preview")
	}
	got, err := p.Complete(input)
	if err != nil || got != first {
		t.Fatalf("Tab disagrees with preview: %q %v", got, err)
	}
	next := p.Suggest(got)
	got, err = p.Complete(got)
	if err != nil || got != next || got == first {
		t.Fatal("preview consumed the completion cycle")
	}
}

func TestDirectorySuggestionAndQuotedUnicodePath(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "耳机 配置"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "耳机.txt"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	input := `"` + filepath.Join(root, "耳")
	p := newPathCompleter(true)
	want := `"` + filepath.Join(root, "耳机 配置") + `\"`
	if got := p.Suggest(input); got != want {
		t.Fatalf("quoted directory preview: %q, want %q", got, want)
	}
	if got := p.Suggest(filepath.Join(root, "missing", "child")); got != "" {
		t.Fatal("missing directory produced a preview")
	}
}

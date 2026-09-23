package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/core"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/i18n"
)

func TestOpenUsesSelectedPathWithoutChangingConfiguration(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	plan, err := core.PrepareInit(root, filepath.Join(t.TempDir(), "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Commit(); err != nil {
		t.Fatal(err)
	}
	store := core.Store{Root: root}
	name := "中文 headphones.txt"
	path := store.ProfilePath(name)
	if err := os.WriteFile(path, []byte("GraphicEQ: 20 0; 1000 -2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	state, err := store.Read()
	if err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []error{nil, errors.New("shell failure")} {
		called := 0
		app := App{Language: i18n.ForLocale("en"), openWith: func(got string) error {
			called++
			if got != path {
				t.Fatalf("wrong path: %q", got)
			}
			return outcome
		}}
		step := app.selectStep("open", store, state)
		transition, err := step.Submit(name)
		if called != 1 {
			t.Fatal("open did not dispatch exactly once")
		}
		if outcome != nil {
			if err == nil {
				t.Fatal("shell error hidden")
			}
		} else if err != nil || transition.Next != nil || transition.Result.Code != 0 || transition.Result.Message != app.Language.Text("openDialogShown", "中文 headphones") {
			t.Fatalf("unexpected open transition: %+v %v", transition, err)
		}
	}
	now, err := store.Read()
	if err != nil || !bytes.Equal(now.Config.Data, state.Config.Data) || now.Current != state.Current {
		t.Fatal("open changed APO state")
	}
}

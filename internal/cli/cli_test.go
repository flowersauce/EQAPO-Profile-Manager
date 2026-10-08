package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/apo"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/core"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/i18n"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/settings"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/ui"
)

func TestUsageAndNonInteractiveCommandsNeverStartWizard(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{
		{nil, 0}, {[]string{"--help"}, 1}, {[]string{"switch", "headphones"}, 1},
		{[]string{"open"}, 1}, {[]string{"init"}, 1}, {[]string{"show"}, 1}, {[]string{"list"}, 3},
	} {
		var out, errOut bytes.Buffer
		a := App{Language: i18n.ForLocale("en"), SettingsPath: filepath.Join(t.TempDir(), "config.json"), Version: "test", Out: &out, Err: &errOut,
			Wizard: func(*ui.Step, i18n.Language) ui.Result { t.Fatal("unexpected interaction"); return ui.Result{} }}
		if got := a.Run(tc.args); got != tc.code {
			t.Errorf("%v: code %d, want %d: %s", tc.args, got, tc.code, errOut.String())
		}
	}
}

func TestOverviewGreetsAccountWithoutControlSequences(t *testing.T) {
	var output bytes.Buffer
	app := App{Language: i18n.ForLocale("en"), Out: &output, Username: "Flower\nname", Version: "1.0.0"}
	app.overview(nil, nil)
	if !strings.HasPrefix(output.String(), "👋 Hi Flower name!\n\nAbout\n") {
		t.Fatalf("unexpected greeting: %q", output.String())
	}
}

func TestInitUsesPlaceholderInsteadOfPrefilledInput(t *testing.T) {
	app := App{Language: i18n.ForLocale("en"), SettingsPath: filepath.Join(t.TempDir(), "config.json")}
	step := app.initStep()
	if step.Initial != "" || step.Placeholder != `C:\Program Files\EqualizerAPO` {
		t.Fatal("init prefilled the input or lost its default")
	}
}

func TestOverviewWithSavedSettingsButUninitializedAPO(t *testing.T) {
	for _, scenario := range []string{"missing root", "missing config", "unmanaged", "missing profiles", "damaged markers"} {
		t.Run(scenario, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "APO")
			if scenario != "missing root" {
				if err := os.MkdirAll(filepath.Join(root, "config"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			if scenario != "missing root" && scenario != "missing config" {
				data := []byte("Preamp: -3 dB\n")
				if scenario == "missing profiles" {
					var err error
					data, err = apo.Takeover(data)
					if err != nil {
						t.Fatal(err)
					}
				}
				if scenario == "damaged markers" {
					data = []byte(apo.Begin + "\n")
				}
				if err := os.WriteFile(filepath.Join(root, "config", "config.txt"), data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			settingsPath := filepath.Join(t.TempDir(), "config.json")
			data, err := json.Marshal(settings.Config{Version: 1, APORoot: root})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(settingsPath, data, 0644); err != nil {
				t.Fatal(err)
			}
			var out, errOut bytes.Buffer
			app := App{Language: i18n.ForLocale("en"), SettingsPath: settingsPath, Out: &out, Err: &errOut, Version: "1.0.0"}
			code := app.Run(nil)
			if scenario == "damaged markers" {
				if code == 0 || errOut.Len() == 0 {
					t.Fatal("damaged configuration was hidden as uninitialized")
				}
				return
			}
			if code != 0 || errOut.Len() != 0 {
				t.Fatalf("overview failed: %d %s", code, errOut.String())
			}
			for _, text := range []string{"About", "Information", "Help", "Not initialized. Run eqm init."} {
				if !strings.Contains(out.String(), text) {
					t.Fatalf("overview is missing %q", text)
				}
			}
			if code := app.Run([]string{"list"}); code == 0 {
				t.Fatal("business command accepted uninitialized APO")
			}
		})
	}
}

func TestAliasesAreCanonicalBeforeWorkerForwarding(t *testing.T) {
	aliases := map[string]string{"im": "import", "in": "init", "l": "list", "o": "open", "p": "print", "rm": "remove", "rn": "rename", "s": "switch"}
	for alias, command := range aliases {
		t.Run(alias, func(t *testing.T) {
			var out, errOut bytes.Buffer
			app := App{Language: i18n.ForLocale("en"), SettingsPath: filepath.Join(t.TempDir(), "config.json"), Out: &out, Err: &errOut, Interactive: true,
				Wizard: func(*ui.Step, i18n.Language) ui.Result { return ui.Result{} }}
			code := app.Run([]string{alias})
			want := 3
			if command == "init" {
				want = 0
			}
			if code != want || len(app.args) != 1 || app.args[0] != command {
				t.Fatalf("alias was not normalized: %d %v %s", code, app.args, errOut.String())
			}
		})
	}
}

func TestRemovedShowCommandIsUnknown(t *testing.T) {
	var out, errOut bytes.Buffer
	app := App{Language: i18n.ForLocale("en"), Out: &out, Err: &errOut, Interactive: true}
	if code := app.Run([]string{"show"}); code != 1 || !strings.Contains(errOut.String(), "Unknown command") {
		t.Fatalf("show was still accepted: %d %s", code, errOut.String())
	}
}

func TestOverviewHelpIsAlphabeticalWithAliases(t *testing.T) {
	var output bytes.Buffer
	app := App{Language: i18n.ForLocale("en"), Out: &output, Version: "test"}
	app.overview(nil, nil)
	help := strings.SplitN(output.String(), "\nHelp\n", 2)[1]
	previous := -1
	for _, command := range []string{"eqm import", "eqm init", "eqm list", "eqm open", "eqm print", "eqm remove", "eqm rename", "eqm switch"} {
		position := strings.Index(help, command)
		if position <= previous {
			t.Fatalf("help is missing or out of order: %s", command)
		}
		previous = position
	}
	if !strings.HasSuffix(help, "^ Aliases: import=im · init=in · list=l · open=o · print=p · remove=rm · rename=rn · switch=s\n") {
		t.Fatal("plain help did not preserve the alias mapping")
	}
}

func flatFixture(t *testing.T) (core.Store, string) {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	plan, err := core.PrepareInit(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := plan.Commit(); err != nil {
		t.Fatal(err)
	}
	return plan.Store, path
}

func TestListIsFlatReadOnlyAndExitsWithoutWizard(t *testing.T) {
	s, path := flatFixture(t)
	for _, name := range []string{"z.txt", "A", "a2.cfg"} {
		if err := os.WriteFile(s.ProfilePath(name), []byte("GraphicEQ: 20 0"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	nested := s.ProfilePath(filepath.Join("旧分类", "hidden.txt"))
	if err := os.Mkdir(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("GraphicEQ: 20 0"), 0644); err != nil {
		t.Fatal(err)
	}
	state, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Switch(state, "A"); err != nil {
		t.Fatal(err)
	}
	state, err = s.Read()
	if err != nil {
		t.Fatal(err)
	}
	for _, interactive := range []bool{false, true} {
		var out, errOut bytes.Buffer
		app := App{Language: i18n.ForLocale("en"), SettingsPath: path, Out: &out, Err: &errOut, Interactive: interactive,
			Wizard: func(*ui.Step, i18n.Language) ui.Result { t.Fatal("list started a wizard"); return ui.Result{} }}
		if code := app.Run([]string{"l"}); code != 0 || errOut.Len() != 0 {
			t.Fatalf("list failed: %d %s", code, errOut.String())
		}
		want := "A\na2.cfg\nz.txt\n"
		if interactive {
			want = "├ A\n├ a2.cfg\n└ z\n"
		}
		if out.String() != want {
			t.Fatalf("unexpected flat listing: %q", out.String())
		}
		if err := state.Config.Check(); err != nil {
			t.Fatal("list changed the configuration")
		}
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatal("list changed a nested profile")
	}
}

func TestImportGoesDirectlyToRootAndStillConfirmsOverwrite(t *testing.T) {
	s, _ := flatFixture(t)
	source := filepath.Join(t.TempDir(), "耳机.txt")
	if err := os.WriteFile(source, []byte("GraphicEQ: 20 0"), 0644); err != nil {
		t.Fatal(err)
	}
	state, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	app := App{Language: i18n.ForLocale("zh-CN")}
	next, err := app.importStep(s, state).Submit(source)
	if err != nil || next.Next == nil || next.Next.Question != app.Language.Text("deleteSourceQuestion") {
		t.Fatalf("import did not finish directly in root: %+v %v", next, err)
	}
	if _, err := os.Stat(s.ProfilePath("耳机.txt")); err != nil {
		t.Fatal("imported file is not at root")
	}
	if _, err := next.Next.Submit("no"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("declining cleanup deleted the source")
	}
	if err := os.WriteFile(source, []byte("GraphicEQ: 20 -3"), 0644); err != nil {
		t.Fatal(err)
	}
	state, err = s.Read()
	if err != nil {
		t.Fatal(err)
	}
	next, err = app.importStep(s, state).Submit(source)
	if err != nil || next.Next == nil || next.Next.Question != app.Language.Text("overwriteQuestion", "耳机") {
		t.Fatalf("overwrite confirmation missing: %+v %v", next, err)
	}
	if _, err := next.Next.Submit("no"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.ProfilePath("耳机.txt"))
	if err != nil || string(data) != "GraphicEQ: 20 0" {
		t.Fatal("declining overwrite changed the profile")
	}
}

func TestMoveAndAliasAreRemoved(t *testing.T) {
	for _, command := range []string{"move", "mv"} {
		var out, errOut bytes.Buffer
		app := App{Language: i18n.ForLocale("en"), Out: &out, Err: &errOut, Interactive: true}
		if code := app.Run([]string{command}); code != 1 || !strings.Contains(errOut.String(), "Unknown command") {
			t.Fatalf("removed command was accepted: %s %d %s", command, code, errOut.String())
		}
	}
}

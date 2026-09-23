package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/apo"
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

package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/core"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/settings"
)

func TestCommitRetriesOnlyDeniedUncommittedOperation(t *testing.T) {
	for _, tc := range []struct {
		name                string
		local               error
		enabled, wantRemote bool
	}{
		{"success", nil, true, false},
		{"permission", os.ErrPermission, true, true},
		{"other error", os.ErrNotExist, true, false},
		{"already elevated", os.ErrPermission, false, false},
		{"partial init", fault.Wrap("initPartial", os.ErrPermission), true, false},
		{"partial rename", fault.Wrap("renamePartial", os.ErrPermission), true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, remote := 0, 0
			sentinel := errors.New("remote outcome")
			request := commitRequest{Operation: "switch", Value: "chosen.txt"}
			app := App{canElevate: tc.enabled, elevate: func(got commitRequest) error {
				remote++
				if got.Operation != request.Operation || got.Value != request.Value {
					t.Fatal("confirmed selection changed")
				}
				return sentinel
			}}
			err := app.commit(request, func() error { calls++; return tc.local })
			if calls != 1 || (remote == 1) != tc.wantRemote || remote > 1 {
				t.Fatalf("local=%d remote=%d", calls, remote)
			}
			if tc.wantRemote && err != sentinel {
				t.Fatal("remote failure was retried or hidden")
			}
			if !tc.wantRemote && err != tc.local {
				t.Fatal("local outcome changed")
			}
		})
	}
}

func TestWorkerExecutesConfirmedInitWithoutWizard(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "config"), 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "config", "config.txt")
	if err := os.WriteFile(configPath, []byte("Preamp: -3 dB\n"), 0644); err != nil {
		t.Fatal(err)
	}
	settingsPath := filepath.Join(t.TempDir(), "config.json")
	plan, err := core.PrepareInit(root, settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	request := commitRequest{Operation: "init", Root: root, Config: &plan.Config, Settings: &plan.Settings}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var restored commitRequest
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if err := restored.execute("switch", settingsPath); err == nil {
		t.Fatal("worker accepted another command")
	}
	if err := restored.execute("init", settingsPath); err != nil {
		t.Fatal(err)
	}
	cfg, err := settings.Load(settingsPath)
	if err != nil || cfg.APORoot != root {
		t.Fatalf("settings were not saved for original user: %+v %v", cfg, err)
	}
	state, err := (core.Store{Root: root}).Read()
	if err != nil || !state.Document.Managed {
		t.Fatalf("init did not complete: %v", err)
	}
}

func TestTransportRejectsChangesWhileUACIsOpen(t *testing.T) {
	for _, replace := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "config.txt")
		if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
			t.Fatal(err)
		}
		snapshot, err := platform.Read(path, false)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(snapshot)
		if err != nil {
			t.Fatal(err)
		}
		if replace {
			// Keep the original file alive so its identity cannot be reused.
			if err := os.Rename(path, path+".old"); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(path, []byte("changed"), 0644); err != nil {
			t.Fatal(err)
		}
		var restored platform.Snapshot
		if err := json.Unmarshal(data, &restored); !fault.IsKey(err, "changed") {
			t.Fatalf("accepted changed confirmation: %v", err)
		}
	}
}

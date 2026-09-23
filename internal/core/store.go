// Package core implements profile operations without terminal dependencies.
package core

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/apo"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/settings"
)

type Store struct{ Root string }
type Profile struct{ Filename string }

func (p Profile) Name() string                 { return p.Filename[:len(p.Filename)-4] }
func (s Store) ConfigDir() string              { return filepath.Join(s.Root, "config") }
func (s Store) ProfilesDir() string            { return filepath.Join(s.ConfigDir(), "eqm-profiles") }
func (s Store) ProfilePath(name string) string { return filepath.Join(s.ProfilesDir(), name) }

type State struct {
	Config   platform.Snapshot
	Document apo.Document
	Profiles []Profile
	Current  string
}

func (s Store) validateDirs() error {
	if err := platform.Directory(s.Root); err != nil {
		return err
	}
	if err := platform.Directory(s.ConfigDir()); err != nil {
		return err
	}
	return nil
}

func (s Store) List() ([]Profile, error) {
	if err := platform.Directory(s.ProfilesDir()); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.ProfilesDir())
	if err != nil {
		return nil, err
	}
	var result []Profile
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.EqualFold(filepath.Ext(entry.Name()), ".txt") {
			continue
		}
		_, err := platform.Regular(s.ProfilePath(entry.Name()))
		if fault.IsKey(err, "regular") {
			continue
		}
		if err != nil {
			return nil, err
		}
		if err := apo.ValidateFilename(entry.Name()); err != nil {
			return nil, fault.Wrap("invalidProfile", err, entry.Name())
		}
		key := strings.ToUpper(entry.Name())
		if seen[key] {
			return nil, fault.New("collision")
		}
		seen[key] = true
		result = append(result, Profile{entry.Name()})
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := strings.ToUpper(result[i].Filename), strings.ToUpper(result[j].Filename)
		if a == b {
			return result[i].Filename < result[j].Filename
		}
		return a < b
	})
	return result, nil
}

func (s Store) Read() (State, error) {
	var state State
	if err := s.validateDirs(); err != nil {
		return state, err
	}
	config, err := platform.Read(filepath.Join(s.ConfigDir(), "config.txt"), false)
	if err != nil {
		return state, err
	}
	doc, err := apo.Parse(config.Data)
	if err != nil {
		return state, err
	}
	if !doc.Managed {
		e := fault.New("uninitialized")
		e.Code = 3
		return state, e
	}
	profiles, err := s.List()
	if err != nil {
		return state, err
	}
	state = State{Config: config, Document: doc, Profiles: profiles}
	if doc.Enabled {
		_, err := platform.Regular(s.ProfilePath(doc.Filename))
		if err == nil {
			state.Current = doc.Filename
		} else if !errors.Is(err, os.ErrNotExist) {
			return state, err
		}
	}
	return state, nil
}

// locked checks the confirmation snapshot, then quietly repairs a missing selection.
func (s Store) locked(state State, action func(State) error) error {
	return platform.WithLock(s.ConfigDir(), func() error {
		if err := s.validateDirs(); err != nil {
			return err
		}
		if err := platform.Directory(s.ProfilesDir()); err != nil {
			return err
		}
		if err := state.Config.Check(); err != nil {
			return err
		}
		fresh, err := s.Read()
		if err != nil {
			return err
		}
		healed := false
		if fresh.Document.Enabled && fresh.Current == "" {
			data, err := fresh.Document.Select(fresh.Document.Filename, false)
			if err != nil {
				return err
			}
			if err := platform.Write(fresh.Config, data); err != nil {
				return err
			}
			healed = true
			fresh, err = s.Read()
			if err != nil {
				return err
			}
		}
		if err := action(fresh); err != nil {
			if healed {
				e := fault.Wrap("healedFailure", err)
				e.Stop = true
				return e
			}
			return err
		}
		return nil
	})
}

func (s Store) Switch(state State, filename string) error {
	return s.locked(state, func(now State) error {
		if filename == "" && !now.Document.Enabled {
			return nil
		}
		target := filename
		if target == "" {
			target = now.Document.Filename
		} else {
			if err := apo.ValidateFilename(target); err != nil {
				return err
			}
			if _, err := platform.Regular(s.ProfilePath(target)); err != nil {
				return err
			}
			if now.Document.Enabled && strings.EqualFold(now.Document.Filename, target) {
				return nil
			}
		}
		data, err := now.Document.Select(target, filename != "")
		if err != nil {
			return err
		}
		return platform.Write(now.Config, data)
	})
}

func (s Store) ReadProfile(filename string) (platform.Snapshot, error) {
	if err := apo.ValidateFilename(filename); err != nil {
		return platform.Snapshot{}, err
	}
	if err := platform.Directory(s.ProfilesDir()); err != nil {
		return platform.Snapshot{}, err
	}
	return platform.Read(s.ProfilePath(filename), false)
}

type InitPlan struct {
	Store            Store
	Config, Settings platform.Snapshot
	Document         apo.Document
}

func PrepareInit(root, settingsPath string) (InitPlan, error) {
	p := InitPlan{}
	clean, err := ParsePath(root)
	if err != nil {
		return p, err
	}
	clean, err = filepath.Abs(clean)
	if err != nil {
		return p, err
	}
	p.Store = Store{Root: clean}
	if err := p.Store.validateDirs(); err != nil {
		return p, fault.Wrap("rootInvalid", err)
	}
	p.Config, err = platform.Read(filepath.Join(p.Store.ConfigDir(), "config.txt"), true)
	if err != nil {
		return p, err
	}
	p.Document, err = apo.Parse(p.Config.Data)
	if err != nil {
		return p, err
	}
	p.Settings, err = platform.Read(settingsPath, true)
	return p, err
}

func (p InitPlan) Commit() error {
	return platform.WithLock(p.Store.ConfigDir(), func() error {
		if err := p.Store.validateDirs(); err != nil {
			return err
		}
		if err := p.Config.Check(); err != nil {
			return err
		}
		if err := p.Settings.Check(); err != nil {
			return err
		}
		if err := platform.ProbeWrite(p.Store.ConfigDir()); err != nil {
			return err
		}
		if p.Config.Info != nil {
			f, err := os.OpenFile(p.Config.Path, os.O_WRONLY, 0)
			if err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
		}
		data, err := apo.Takeover(p.Config.Data)
		if err != nil {
			return err
		}
		if p.Document.Enabled {
			if _, err := platform.Regular(p.Store.ProfilePath(p.Document.Filename)); errors.Is(err, os.ErrNotExist) {
				data, err = p.Document.Select(p.Document.Filename, false)
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
		if err := os.MkdirAll(p.Store.ProfilesDir(), 0755); err != nil {
			return err
		}
		if err := platform.Directory(p.Store.ProfilesDir()); err != nil {
			return err
		}
		if err := platform.Write(p.Config, data); err != nil {
			return err
		}
		if err := settings.Save(p.Settings, p.Store.Root); err != nil {
			return fault.Wrap("initPartial", err)
		}
		return nil
	})
}

// ParsePath preserves unquoted spaces and accepts one outer quote pair.
func ParsePath(input string) (string, error) {
	input = strings.TrimSpace(input)
	if strings.ContainsAny(input, "\r\n\x00") || input == "" {
		return "", fault.New("onePath")
	}
	if strings.HasPrefix(input, "\"") {
		if !strings.HasSuffix(input, "\"") || len(input) < 2 {
			return "", fault.New("onePath")
		}
		input = input[1 : len(input)-1]
	} else if strings.HasPrefix(input, "'") && strings.HasSuffix(input, "'") && len(input) > 1 {
		input = input[1 : len(input)-1]
	}
	if input == "" || strings.Contains(input, "\"") {
		return "", fault.New("onePath")
	}
	// Multiple absolute drive paths cannot form a valid Windows filename.
	if strings.Count(input, ":") > 1 || strings.Contains(input, ` \\`) {
		return "", fault.New("onePath")
	}
	return filepath.Clean(input), nil
}

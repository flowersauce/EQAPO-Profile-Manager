package core

import (
	"bytes"
	"errors"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/apo"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
)

var parametric = regexp.MustCompile(`(?i)^Filter\s+\d+:\s+ON\s+(PK|LS|HS|LSC|HSC)\s+Fc\s+([+-]?[0-9.]+)\s+Hz\s+Gain\s+([+-]?[0-9.]+)\s+dB(?:\s+Q\s+([0-9.]+))?\s*$`)

// CheckFormat is a lightweight AutoEQ signature check, not a complete APO parser.
func CheckFormat(data []byte) error {
	bad := func() error { e := fault.New("format"); e.Code = 4; return e }
	if !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
		return bad()
	}
	graphic, filters := false, 0
	for _, line := range strings.Split(strings.TrimPrefix(string(data), "\xef\xbb\xbf"), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "preamp:") {
			parts := strings.Fields(strings.TrimSpace(line[len("Preamp:"):]))
			if len(parts) != 2 || !strings.EqualFold(parts[1], "dB") {
				return bad()
			}
			if !finite(parts[0]) {
				return bad()
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "graphiceq:") {
			for _, pair := range strings.Split(line[len("GraphicEQ:"):], ";") {
				fields := strings.Fields(pair)
				if len(fields) != 2 {
					return bad()
				}
				freq, e1 := strconv.ParseFloat(fields[0], 64)
				_, e2 := strconv.ParseFloat(fields[1], 64)
				if e1 != nil || e2 != nil || freq <= 0 || !finite(fields[0]) || !finite(fields[1]) {
					return bad()
				}
			}
			graphic = true
			continue
		}
		m := parametric.FindStringSubmatch(line)
		if m == nil {
			return bad()
		}
		freq, e1 := strconv.ParseFloat(m[2], 64)
		_, e2 := strconv.ParseFloat(m[3], 64)
		if e1 != nil || e2 != nil || freq <= 0 {
			return bad()
		}
		if m[4] != "" {
			q, err := strconv.ParseFloat(m[4], 64)
			if err != nil || q <= 0 {
				return bad()
			}
		}
		filters++
	}
	if !graphic && filters == 0 {
		return bad()
	}
	return nil
}

func finite(value string) bool {
	f, err := strconv.ParseFloat(value, 64)
	return err == nil && !math.IsNaN(f) && !math.IsInf(f, 0)
}

type ImportPlan struct {
	Source, Target platform.Snapshot
	Filename       string
}

func (s Store) PrepareImport(input string) (ImportPlan, error) {
	p := ImportPlan{}
	path, err := ParsePath(input)
	if err != nil {
		return p, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return p, err
	}
	p.Filename = filepath.Base(path)
	if err := apo.ValidateFilename(p.Filename); err != nil {
		return p, err
	}
	p.Source, err = platform.Read(path, false)
	if err != nil {
		return p, err
	}
	if err := CheckFormat(p.Source.Data); err != nil {
		return p, err
	}
	profiles, err := s.List()
	if err != nil {
		return p, err
	}
	for _, profile := range profiles {
		if strings.EqualFold(profile.Filename, p.Filename) {
			p.Filename = profile.Filename
			break
		}
	}
	p.Target, err = platform.Read(s.ProfilePath(p.Filename), true)
	if err != nil {
		return p, err
	}
	if p.Target.Info != nil && os.SameFile(p.Source.Info, p.Target.Info) {
		return p, fault.New("sameFile")
	}
	return p, nil
}

func (s Store) Import(state State, p ImportPlan) error {
	return s.locked(state, func(now State) error {
		for _, profile := range now.Profiles {
			if strings.EqualFold(profile.Filename, p.Filename) && profile.Filename != p.Filename {
				return fault.New("changed", p.Target.Path)
			}
		}
		if err := p.Source.Check(); err != nil {
			return err
		}
		if err := p.Target.Check(); err != nil {
			return err
		}
		return platform.Write(p.Target, p.Source.Data)
	})
}

func (p ImportPlan) DeleteSource() error {
	if err := p.Source.Check(); err != nil {
		return fault.Wrap("sourceKept", err)
	}
	// Source deletion is only allowed after a matching import is on disk.
	target, err := platform.Read(p.Target.Path, false)
	if err != nil {
		return fault.Wrap("sourceKept", err)
	}
	if !bytes.Equal(target.Data, p.Source.Data) || os.SameFile(target.Info, p.Source.Info) {
		return fault.New("sourceKept")
	}
	if err := os.Remove(p.Source.Path); err != nil {
		return fault.Wrap("sourceKept", err)
	}
	return nil
}

func (s Store) Rename(state State, source platform.Snapshot, stem string) error {
	name := stem + filepath.Ext(source.Path)
	if err := apo.ValidateFilename(name); err != nil {
		return err
	}
	return s.locked(state, func(now State) error {
		if err := source.Check(); err != nil {
			return err
		}
		old := filepath.Base(source.Path)
		if old == name {
			return nil
		}
		profiles, err := s.List()
		if err != nil {
			return err
		}
		for _, p := range profiles {
			if strings.EqualFold(p.Filename, name) && p.Filename != old {
				e := fault.New("collision")
				e.ClearInput = true
				return e
			}
		}
		target := s.ProfilePath(name)
		if err := platform.Rename(source.Path, target); err != nil {
			return err
		}
		if strings.EqualFold(now.Document.Filename, old) {
			data, err := now.Document.Select(name, now.Document.Enabled)
			if err == nil {
				err = platform.Write(now.Config, data)
			}
			if err != nil {
				if rollback := platform.Rename(target, source.Path); rollback != nil {
					return fault.Wrap("renamePartial", errors.Join(err, rollback), target)
				}
				return err
			}
		}
		return nil
	})
}

func (s Store) Remove(state State, source platform.Snapshot) error {
	return s.locked(state, func(now State) error {
		if err := source.Check(); err != nil {
			return err
		}
		changed := now.Document.Enabled && strings.EqualFold(now.Document.Filename, filepath.Base(source.Path))
		var disabled platform.Snapshot
		if changed {
			data, err := now.Document.Select(now.Document.Filename, false)
			if err != nil {
				return err
			}
			if err := platform.Write(now.Config, data); err != nil {
				return err
			}
			disabled, err = platform.Read(now.Config.Path, false)
			if err != nil {
				return fault.Wrap("removePartial", err)
			}
		}
		if err := os.Remove(source.Path); err != nil {
			if changed {
				if rollback := platform.Write(disabled, now.Config.Data); rollback != nil {
					return fault.Wrap("removePartial", errors.Join(err, rollback))
				}
			}
			return err
		}
		return nil
	})
}

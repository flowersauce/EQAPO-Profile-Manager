// Package settings separates portable and installed user settings.
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
)

type Config struct {
	Version int    `json:"version"`
	APORoot string `json:"apo_root"`
}

func Path() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	local, err := platform.LocalAppData()
	if err != nil {
		return "", err
	}
	return PathFor(exe, local)
}

// PathFor performs no writes. localAppData belongs to the original invoking user.
func PathFor(exe, localAppData string) (string, error) {
	if !filepath.IsAbs(exe) || !filepath.IsAbs(localAppData) {
		return "", fault.New("settingsLocation")
	}
	flag := filepath.Join(filepath.Dir(exe), "portable.flag")
	info, err := os.Lstat(flag)
	if err == nil {
		if !info.Mode().IsRegular() {
			return "", fault.New("portableFlag")
		}
		return filepath.Join(filepath.Dir(exe), "config", "config.json"), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return filepath.Join(localAppData, "EQM", "config.json"), nil
}

func Load(path string) (Config, error) {
	var cfg Config
	s, err := platform.Read(path, true)
	if err != nil {
		return cfg, fault.Wrap("settings", err)
	}
	if s.Info == nil {
		e := fault.New("uninitialized")
		e.Code = 3
		return cfg, e
	}
	if err := json.Unmarshal(s.Data, &cfg); err != nil {
		e := fault.Wrap("settings", err)
		e.Code = 3
		return cfg, e
	}
	if cfg.Version != 1 || !filepath.IsAbs(cfg.APORoot) {
		e := fault.New("settings")
		e.Code = 3
		return cfg, e
	}
	return cfg, nil
}

func Save(snapshot platform.Snapshot, root string) error {
	dir := filepath.Dir(snapshot.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := platform.Directory(dir); err != nil {
		return err
	}
	data, err := json.MarshalIndent(Config{Version: 1, APORoot: root}, "", "  ")
	if err != nil {
		return err
	}
	return platform.WithLock(dir, func() error {
		if err := platform.ProbeWrite(dir); err != nil {
			return err
		}
		return platform.Write(snapshot, append(data, '\n'))
	})
}

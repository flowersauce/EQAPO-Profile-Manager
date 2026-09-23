package platform

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
)

// Snapshot records identity and bytes so confirmation applies to the file read.
type Snapshot struct {
	Path     string
	Data     []byte
	Info     os.FileInfo
	identity [3]uint32
}

// ProbeWrite is used only after init has been submitted, never during preview.
func ProbeWrite(dir string) error {
	f, err := os.CreateTemp(dir, ".eqm-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	return os.Remove(name)
}

func Read(path string, allowMissing bool) (Snapshot, error) {
	s := Snapshot{Path: path}
	info, err := Regular(path)
	if allowMissing && errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	f, err := os.Open(path)
	if err != nil {
		return s, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return s, err
	}
	if !os.SameFile(info, opened) {
		return s, fault.New("changed", path)
	}
	s.Data, err = io.ReadAll(f)
	if err != nil {
		return s, err
	}
	s.Info = opened
	s.identity, err = fileIdentity(f)
	return s, err
}

func (s Snapshot) Check() error {
	now, err := Read(s.Path, true)
	if err != nil {
		return err
	}
	if s.Info == nil && now.Info == nil {
		return nil
	}
	if s.Info == nil || now.Info == nil || !os.SameFile(s.Info, now.Info) || !bytes.Equal(s.Data, now.Data) {
		return fault.New("changed", s.Path)
	}
	return nil
}

// Write replaces one complete file. It never deletes the destination first.
func Write(expected Snapshot, data []byte) error {
	if err := expected.Check(); err != nil {
		return err
	}
	if expected.Info != nil && bytes.Equal(expected.Data, data) {
		return nil
	}
	f, err := os.CreateTemp(filepath.Dir(expected.Path), ".eqm-*.tmp")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = expected.Check(); err != nil {
		return err
	}
	return replaceFile(temp, expected.Path, expected.Info != nil)
}

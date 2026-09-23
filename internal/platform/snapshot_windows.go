//go:build windows

package platform

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"golang.org/x/sys/windows"
)

func fileIdentity(f *os.File) ([3]uint32, error) {
	var info windows.ByHandleFileInformation
	err := windows.GetFileInformationByHandle(windows.Handle(f.Fd()), &info)
	return [3]uint32{info.VolumeSerialNumber, info.FileIndexHigh, info.FileIndexLow}, err
}

type snapshotWire struct {
	Path     string
	Data     []byte
	Exists   bool
	Identity [3]uint32
}

// MarshalJSON transfers the original confirmation, including file identity.
func (s Snapshot) MarshalJSON() ([]byte, error) {
	return json.Marshal(snapshotWire{s.Path, s.Data, s.Info != nil, s.identity})
}

// UnmarshalJSON restores a snapshot only if the confirmed file is unchanged.
// The worker then uses the normal core checks and locks at commit time.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	var wire snapshotWire
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if !filepath.IsAbs(wire.Path) {
		return fault.New("elevationContext")
	}
	now, err := Read(wire.Path, true)
	if err != nil {
		return err
	}
	if (now.Info != nil) != wire.Exists || now.identity != wire.Identity || !bytes.Equal(now.Data, wire.Data) {
		return fault.New("changed", wire.Path)
	}
	*s = now
	return nil
}

//go:build windows

package platform

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"runtime"
	"unsafe"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"golang.org/x/sys/windows"
)

// WorkerPipe is a private, duplex pipe bound to the other process's lifetime.
// Calls are serialized by the wizard; neither endpoint exposes a shell.
type WorkerPipe struct {
	pipe, peer windows.Handle
}

func (p *WorkerPipe) Close() {
	if p == nil {
		return
	}
	if p.pipe != 0 {
		windows.CloseHandle(p.pipe)
		p.pipe = 0
	}
	if p.peer != 0 {
		windows.CloseHandle(p.peer)
		p.peer = 0
	}
}

// await always drains cancelled I/O before releasing OVERLAPPED and its buffer.
func (p *WorkerPipe) await(ov *windows.Overlapped, err error, timeout uint32) (uint32, error) {
	if err != nil && !errors.Is(err, windows.ERROR_IO_PENDING) {
		return 0, err
	}
	if errors.Is(err, windows.ERROR_IO_PENDING) {
		status, waitErr := windows.WaitForMultipleObjects([]windows.Handle{ov.HEvent, p.peer}, false, timeout)
		if waitErr != nil || status != windows.WAIT_OBJECT_0 {
			_ = windows.CancelIoEx(p.pipe, ov)
			var drained uint32
			_ = windows.GetOverlappedResult(p.pipe, ov, &drained, true)
			if waitErr != nil {
				return 0, waitErr
			}
			return 0, fault.New("elevationDisconnected")
		}
	}
	var count uint32
	err = windows.GetOverlappedResult(p.pipe, ov, &count, false)
	return count, err
}

func (p *WorkerPipe) transfer(data []byte, write bool) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return 0, err
	}
	defer windows.CloseHandle(event)
	ov := windows.Overlapped{HEvent: event}
	var count uint32
	if write {
		err = windows.WriteFile(p.pipe, data, &count, &ov)
	} else {
		err = windows.ReadFile(p.pipe, data, &count, &ov)
	}
	count, err = p.await(&ov, err, windows.INFINITE)
	runtime.KeepAlive(data)
	if errors.Is(err, windows.ERROR_BROKEN_PIPE) || (err == nil && count == 0) {
		return 0, io.EOF
	}
	return int(count), err
}

func (p *WorkerPipe) Read(data []byte) (int, error)  { return p.transfer(data, false) }
func (p *WorkerPipe) Write(data []byte) (int, error) { return p.transfer(data, true) }

// Length framing bounds allocations and prevents a JSON decoder reading ahead.
const maxWorkerMessage = 32 << 20

func (p *WorkerPipe) Send(data []byte) error {
	if len(data) > maxWorkerMessage {
		return fault.New("elevationContext")
	}
	frame := make([]byte, 4+len(data))
	binary.LittleEndian.PutUint32(frame, uint32(len(data)))
	copy(frame[4:], data)
	for len(frame) > 0 {
		n, err := p.Write(frame)
		if err != nil {
			return err
		}
		frame = frame[n:]
	}
	return nil
}

func (p *WorkerPipe) Receive() ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(p, header[:]); err != nil {
		return nil, err
	}
	size := binary.LittleEndian.Uint32(header[:])
	if size > maxWorkerMessage {
		return nil, fault.New("elevationContext")
	}
	data := make([]byte, int(size))
	_, err := io.ReadFull(p, data)
	return data, err
}

func StartWorker(args []string, context ElevationContext) (_ *WorkerPipe, resultErr error) {
	tokenUser, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, err
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;;GA;;;" + tokenUser.User.Sid.String() + ")(A;;GA;;;BA)")
	if err != nil {
		return nil, err
	}
	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	context.Pipe = `\\.\pipe\eqm-` + hex.EncodeToString(nonce[:])
	context.Parent = uint32(os.Getpid())
	name, err := windows.UTF16PtrFromString(context.Pipe)
	if err != nil {
		return nil, err
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: sd}
	handle, err := windows.CreateNamedPipe(name, windows.PIPE_ACCESS_DUPLEX|windows.FILE_FLAG_OVERLAPPED|windows.FILE_FLAG_FIRST_PIPE_INSTANCE,
		windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT|windows.PIPE_REJECT_REMOTE_CLIENTS, 1, 65536, 65536, 0, &sa)
	if err != nil {
		return nil, err
	}
	p := &WorkerPipe{pipe: handle}
	defer func() {
		if resultErr != nil {
			p.Close()
		}
	}()
	p.peer, err = startElevated(args, context)
	if err != nil {
		return nil, err
	}
	event, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(event)
	ov := windows.Overlapped{HEvent: event}
	err = windows.ConnectNamedPipe(p.pipe, &ov)
	if !errors.Is(err, windows.ERROR_PIPE_CONNECTED) {
		if _, err = p.await(&ov, err, 30000); err != nil {
			return nil, err
		}
	}
	var client uint32
	if err := windows.GetNamedPipeClientProcessId(p.pipe, &client); err != nil {
		return nil, err
	}
	expected, err := windows.GetProcessId(p.peer)
	if err != nil {
		return nil, err
	}
	if client != expected {
		return nil, fault.New("elevationContext")
	}
	return p, nil
}

func ConnectWorker(context ElevationContext) (_ *WorkerPipe, resultErr error) {
	if !Elevated() {
		return nil, fault.New("elevationContext")
	}
	name, err := windows.UTF16PtrFromString(context.Pipe)
	if err != nil {
		return nil, err
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, err
	}
	p := &WorkerPipe{pipe: handle}
	defer func() {
		if resultErr != nil {
			p.Close()
		}
	}()
	var server uint32
	if err := windows.GetNamedPipeServerProcessId(p.pipe, &server); err != nil {
		return nil, err
	}
	if server != context.Parent {
		return nil, fault.New("elevationContext")
	}
	p.peer, err = windows.OpenProcess(windows.SYNCHRONIZE, false, server)
	if err != nil {
		return nil, err
	}
	return p, nil
}

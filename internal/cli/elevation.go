package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowersauce/EQAPO-Profile-Manager/internal/apo"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/core"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/i18n"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/settings"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/ui"
)

// commitRequest describes exactly one already-confirmed business operation.
// Snapshot decoding in the worker rejects changed bytes or file identities.
type commitRequest struct {
	Operation, Root, Value, TargetPath string
	Config, Settings, Source, Target   *platform.Snapshot
}

type commitResponse struct {
	Message                   string
	Code                      int
	Stop, ClearInput, Warning bool
}

func (a *App) commit(request commitRequest, local func() error) error {
	err := local()
	if !platform.PermissionDenied(err) || !a.canElevate {
		return err
	}
	// Do not replay a transaction that already committed some of its changes.
	var detail *fault.Error
	if errors.As(err, &detail) && detail.Stop {
		return err
	}
	return a.elevate(request)
}

func (a *App) commitElevated(request commitRequest) error {
	if a.worker == nil {
		select {
		case a.progress <- ui.Notice{Text: a.Language.Text("elevationRequired"), Warning: true}:
		default:
		}
		worker, err := platform.StartWorker(a.args, a.context)
		if err != nil {
			if fault.IsKey(err, "elevationCancelled") {
				return err
			}
			return fault.Wrap("elevationFailed", err)
		}
		a.worker = worker
		select {
		case a.progress <- ui.Notice{Text: a.Language.Text("working")}:
		default:
		}
	}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if err := a.worker.Send(data); err != nil {
		return workerDisconnected(err)
	}
	data, err = a.worker.Receive()
	if err != nil {
		return workerDisconnected(err)
	}
	var response commitResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return workerDisconnected(err)
	}
	if response.Message == "" {
		return nil
	}
	failure := fault.New("workerError", response.Message)
	failure.Code, failure.Stop, failure.ClearInput = response.Code, response.Stop, response.ClearInput
	failure.Warning = response.Warning
	return failure
}

func workerDisconnected(err error) error {
	e := fault.Wrap("elevationDisconnected", err)
	e.Stop = true // Outcome may be unknown; never replay automatically.
	return e
}

func runWorker(context platform.ElevationContext, args []string) int {
	if len(args) != 1 {
		return 1
	}
	switch args[0] {
	case "init", "switch", "import", "remove", "rename":
	default:
		return 1
	}
	pipe, err := platform.ConnectWorker(context)
	if err != nil {
		return 1
	}
	defer pipe.Close()
	exe, err := os.Executable()
	if err != nil {
		return 1
	}
	settingsPath, err := settings.PathFor(exe, context.LocalAppData)
	if err != nil {
		return 1
	}
	lang := i18n.ForLocale(context.Locale)
	for {
		data, err := pipe.Receive()
		if err != nil {
			return 0
		} // Parent closed or exited; no persistent service.
		var request commitRequest
		err = json.Unmarshal(data, &request)
		if err == nil {
			err = request.execute(args[0], settingsPath)
		}
		response := commitResponse{}
		if err != nil {
			response.Code = fault.ExitCode(err)
			var detail *fault.Error
			if errors.As(err, &detail) {
				response.Stop, response.ClearInput = detail.Stop, detail.ClearInput
				response.Warning = detail.Warning
			}
			if platform.PermissionDenied(err) {
				err = fault.Wrap("elevationDenied", err)
				response.Stop = true
			}
			response.Message = lang.Error(err)
		}
		data, err = json.Marshal(response)
		if err != nil || pipe.Send(data) != nil {
			return 1
		}
	}
}

func samePath(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }

func (r commitRequest) execute(command, settingsPath string) error {
	if r.Operation != command && !(command == "import" && r.Operation == "deleteSource") {
		return fault.New("elevationContext")
	}
	if !filepath.IsAbs(r.Root) {
		return fault.New("elevationContext")
	}
	store := core.Store{Root: r.Root}
	if r.Operation == "deleteSource" {
		if r.Source == nil || !samePath(filepath.Dir(r.TargetPath), store.ProfilesDir()) {
			return fault.New("elevationContext")
		}
		return (core.ImportPlan{Source: *r.Source, Target: platform.Snapshot{Path: r.TargetPath}}).DeleteSource()
	}
	if r.Config == nil || !samePath(r.Config.Path, filepath.Join(store.ConfigDir(), "config.txt")) {
		return fault.New("elevationContext")
	}
	state := core.State{Config: *r.Config}
	switch r.Operation {
	case "init":
		if r.Settings == nil || !samePath(r.Settings.Path, settingsPath) {
			return fault.New("elevationContext")
		}
		doc, err := apo.Parse(r.Config.Data)
		if err != nil {
			return err
		}
		return (core.InitPlan{Store: store, Config: *r.Config, Settings: *r.Settings, Document: doc}).Commit()
	case "switch":
		return store.Switch(state, r.Value)
	case "import":
		if r.Source == nil || r.Target == nil {
			return fault.New("elevationContext")
		}
		if err := apo.ValidateFilename(r.Value); err != nil {
			return err
		}
		if !samePath(r.Target.Path, store.ProfilePath(r.Value)) {
			return fault.New("elevationContext")
		}
		if err := core.CheckFormat(r.Source.Data); err != nil {
			return err
		}
		return store.Import(state, core.ImportPlan{Source: *r.Source, Target: *r.Target, Filename: r.Value})
	case "rename", "remove":
		if r.Source == nil || !samePath(filepath.Dir(r.Source.Path), store.ProfilesDir()) {
			return fault.New("elevationContext")
		}
		if err := apo.ValidateFilename(filepath.Base(r.Source.Path)); err != nil {
			return err
		}
		if r.Operation == "rename" {
			return store.Rename(state, *r.Source, r.Value)
		}
		return store.Remove(state, *r.Source)
	}
	return fault.New("elevationContext")
}

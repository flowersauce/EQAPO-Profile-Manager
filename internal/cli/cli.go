// Package cli maps commands to localized business flows.
package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/core"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/i18n"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/settings"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/ui"
)

type App struct {
	Language              i18n.Language
	SettingsPath, Version string
	Username              string
	Out, Err              io.Writer
	Interactive           bool
	Wizard                func(*ui.Step, i18n.Language) ui.Result
	canElevate            bool
	worker                *platform.WorkerPipe
	args                  []string
	context               platform.ElevationContext
	elevate               func(commitRequest) error
	progress              chan ui.Notice
	openWith              func(string) error
}

var commands = []struct{ name, key string }{
	{"switch", "commandSwitch"}, {"list", "commandList"}, {"import", "commandImport"},
	{"remove", "commandRemove"}, {"show", "commandShow"}, {"open", "commandOpen"}, {"rename", "commandRename"}, {"init", "commandInit"},
}

func Main(args []string, version string) int {
	locale := platform.UILanguage()
	context, commandArgs, child, err := platform.DecodeElevation(args, platform.Elevated())
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.OutputTheme(os.Stderr).Feedback(i18n.ForLocale(locale).Error(err), fault.IsWarning(err)))
		return 1
	}
	if child {
		return runWorker(context, commandArgs)
	} else {
		context.Locale = locale
		context.LocalAppData, err = platform.LocalAppData()
		if err != nil {
			fmt.Fprintln(os.Stderr, ui.OutputTheme(os.Stderr).Feedback(i18n.ForLocale(locale).Error(err), fault.IsWarning(err)))
			return 1
		}
	}
	lang := i18n.ForLocale(locale)
	restore, err := platform.UTF8Console()
	defer restore()
	if err != nil {
		fmt.Fprintln(os.Stderr, ui.OutputTheme(os.Stderr).Feedback(lang.Error(err), fault.IsWarning(err)))
		return 1
	}
	a := &App{Language: lang, Version: version, Out: os.Stdout, Err: os.Stderr,
		Interactive: platform.IsTerminal(os.Stdin) && platform.IsTerminal(os.Stdout) && platform.IsTerminal(os.Stderr), Wizard: ui.Run}
	a.canElevate = a.Interactive && !platform.Elevated()
	a.args, a.context = commandArgs, context
	a.elevate = a.commitElevated
	a.openWith = platform.OpenWith
	a.Username = platform.Username()
	a.Wizard = func(step *ui.Step, lang i18n.Language) ui.Result {
		a.progress = make(chan ui.Notice, 1)
		return ui.RunWithStatus(step, lang, a.progress)
	}
	defer func() { a.worker.Close() }()
	exe, err := os.Executable()
	if err == nil {
		a.SettingsPath, err = settings.PathFor(exe, context.LocalAppData)
	}
	var code int
	if err != nil {
		code = a.fail(err)
	} else {
		code = a.Run(commandArgs)
	}
	return code
}

func (a *App) Run(args []string) int {
	command := ""
	if len(args) > 1 {
		return a.fail(fault.New("usage"))
	}
	if len(args) == 1 {
		command = args[0]
		known := false
		for _, c := range commands {
			if command == c.name {
				known = true
				break
			}
		}
		if !known {
			return a.fail(fault.New("usage"))
		}
	}
	if command != "" && command != "list" && !a.Interactive {
		return a.fail(fault.New("tty"))
	}
	if command == "init" {
		return a.runWizard(a.initStep())
	}
	cfg, err := settings.Load(a.SettingsPath)
	if err != nil {
		if command == "" && fault.IsKey(err, "uninitialized") {
			a.overview(nil, nil)
			return 0
		}
		return a.fail(err)
	}
	s := core.Store{Root: cfg.APORoot}
	state, err := s.Read()
	if err != nil {
		if command == "" && (fault.IsKey(err, "uninitialized") || errors.Is(err, os.ErrNotExist)) {
			a.overview(nil, nil)
			return 0
		}
		return a.fail(err)
	}
	switch command {
	case "":
		a.overview(&s, &state)
		return 0
	case "list":
		if len(state.Profiles) == 0 {
			fmt.Fprintln(a.Out, ui.OutputTheme(a.Out).Hint(a.Language.Text("empty")))
			return 0
		}
		for _, p := range state.Profiles {
			fmt.Fprintln(a.Out, i18n.Safe(p.Name()))
		}
		return 0
	case "import":
		return a.runWizard(a.importStep(s, state))
	default:
		if command != "switch" && len(state.Profiles) == 0 {
			fmt.Fprintln(a.Out, ui.OutputTheme(a.Out).Hint(a.Language.Text("empty")))
			return 0
		}
		return a.runWizard(a.selectStep(command, s, state))
	}
}

func (a *App) fail(err error) int {
	if platform.PermissionDenied(err) {
		if !a.Interactive {
			err = fault.New("elevationTerminal")
		} else {
			err = fault.Wrap("elevationDenied", err)
		}
	}
	fmt.Fprintln(a.Err, ui.OutputTheme(a.Err).Feedback(a.Language.Error(err), fault.IsWarning(err)))
	return fault.ExitCode(err)
}

func (a *App) runWizard(step *ui.Step) int {
	result := a.Wizard(step, a.Language)
	if result.Err != nil {
		return a.fail(result.Err)
	}
	if result.Message != "" {
		theme := ui.OutputTheme(a.Err)
		message := theme.Success("✓ " + i18n.Safe(result.Message))
		if result.Code != 0 {
			message = theme.Feedback(i18n.Safe(result.Message), true)
		}
		fmt.Fprintln(a.Err, message)
	}
	if result.Data != nil {
		if _, err := a.Out.Write(result.Data); err != nil {
			return a.fail(err)
		}
	}
	return result.Code
}

func (a *App) overview(s *core.Store, state *core.State) {
	theme := ui.OutputTheme(a.Out)
	if a.Username == "" {
		fmt.Fprintln(a.Out, "👋 Hi!")
	} else {
		fmt.Fprintf(a.Out, "👋 Hi %s!\n", i18n.Safe(a.Username))
	}
	fmt.Fprintln(a.Out)
	section := func(key string) {
		fmt.Fprintln(a.Out, theme.Heading(a.Language.Text(key)))
	}
	branch := func(index, count int) string {
		if index == count-1 {
			return theme.Muted("└")
		}
		return theme.Muted("├")
	}
	// Pad by display columns so Chinese and English labels share a value column.
	rows := func(entries [][2]string) {
		width := 0
		for _, row := range entries {
			width = max(width, ansi.StringWidth(row[0]))
		}
		for i, row := range entries {
			padding := strings.Repeat(" ", width-ansi.StringWidth(row[0]))
			fmt.Fprintf(a.Out, "%s %s%s %s %s\n", branch(i, len(entries)), row[0], padding, theme.Muted("·"), row[1])
		}
	}
	section("aboutHeading")
	rows([][2]string{
		{a.Language.Text("version"), i18n.Safe(a.Version)},
		{a.Language.Text("author"), "flowersauce"},
		{a.Language.Text("repository"), theme.Link("https://github.com/flowersauce/EQAPO-Profile-Manager")},
	})
	fmt.Fprintln(a.Out)
	section("infoHeading")
	if state == nil {
		fmt.Fprintln(a.Out, branch(0, 1)+" "+theme.Warning(a.Language.Text("uninitialized")))
	} else {
		current := theme.Warning("None")
		if state.Current != "" {
			current = i18n.Safe(displayName(state.Current))
		}
		rows([][2]string{
			{a.Language.Text("status"), current},
			{a.Language.Text("count"), fmt.Sprint(len(state.Profiles))},
			{a.Language.Text("apoDir"), i18n.Safe(s.Root)},
		})
	}
	fmt.Fprintln(a.Out)
	section("helpHeading")
	for i, c := range commands {
		fmt.Fprintf(a.Out, "%s %s %s%s %s %s\n", branch(i, len(commands)), theme.Muted("eqm"), theme.Focus(c.name), strings.Repeat(" ", 6-len(c.name)), theme.Muted("·"), a.Language.Text(c.key))
	}
}

func displayName(filename string) string { return strings.TrimSuffix(filename, filepath.Ext(filename)) }

func (a *App) done(key string, args ...any) ui.Transition {
	return ui.Transition{Result: ui.Result{Message: a.Language.Text(key, args...)}}
}
func (a *App) cancelled() ui.Transition {
	return ui.Transition{Result: ui.Result{Code: 2, Message: a.Language.Text("cancelled")}}
}

func (a *App) confirm(question string, submit func(bool) (ui.Transition, error)) *ui.Step {
	return &ui.Step{Question: question, Confirm: true, Selected: 1,
		Options: []ui.Option{{Label: a.Language.Text("yes"), Value: "yes"}, {Label: a.Language.Text("no"), Value: "no"}},
		Submit:  func(value string) (ui.Transition, error) { return submit(value == "yes") }}
}

func (a *App) initStep() *ui.Step {
	completion := newPathCompleter(true)
	initial := `C:\Program Files\EqualizerAPO`
	if cfg, err := settings.Load(a.SettingsPath); err == nil {
		initial = cfg.APORoot
	}
	return &ui.Step{Question: a.Language.Text("rootQuestion"), Help: a.Language.Text("initPathHelp"), Placeholder: initial, Complete: completion.Complete, Suggest: completion.Suggest,
		Submit: func(root string) (ui.Transition, error) {
			plan, err := core.PrepareInit(root, a.SettingsPath)
			if err != nil {
				return ui.Transition{}, err
			}
			if plan.Document.Managed {
				if err := a.commit(commitRequest{Operation: "init", Root: plan.Store.Root, Config: &plan.Config, Settings: &plan.Settings}, plan.Commit); err != nil {
					return ui.Transition{}, err
				}
				return a.done("initDone"), nil
			}
			step := a.confirm(a.Language.Text("takeoverQuestion"), func(yes bool) (ui.Transition, error) {
				if !yes {
					return a.cancelled(), nil
				}
				if err := a.commit(commitRequest{Operation: "init", Root: plan.Store.Root, Config: &plan.Config, Settings: &plan.Settings}, plan.Commit); err != nil {
					return ui.Transition{}, err
				}
				return a.done("initDone"), nil
			})
			step.Help = a.Language.Text("takeoverHelp", plan.Config.Path)
			return ui.Transition{Next: step}, nil
		}}
}

func (a *App) selectStep(command string, s core.Store, state core.State) *ui.Step {
	keys := map[string]string{"switch": "selectSwitch", "show": "selectShow", "open": "selectOpen", "rename": "selectRename", "remove": "selectRemove"}
	step := &ui.Step{Question: a.Language.Text(keys[command])}
	if command == "switch" {
		step.Help = a.Language.Text("chooseHelp") + " · " + a.Language.Text("appliedLegend")
		step.Options = append(step.Options, ui.Option{Label: "None", Value: "", Current: state.Current == ""})
	}
	for _, p := range state.Profiles {
		label := p.Name()
		current := command == "switch" && strings.EqualFold(state.Current, p.Filename)
		if current {
			step.Selected = len(step.Options)
		}
		step.Options = append(step.Options, ui.Option{Label: label, Value: p.Filename, Current: current})
	}
	step.Submit = func(filename string) (ui.Transition, error) {
		if command == "switch" {
			if err := a.commit(commitRequest{Operation: "switch", Root: s.Root, Config: &state.Config, Value: filename}, func() error { return s.Switch(state, filename) }); err != nil {
				return ui.Transition{}, err
			}
			name := "None"
			if filename != "" {
				name = displayName(filename)
			}
			return a.done("switched", name), nil
		}
		profile, err := s.ReadProfile(filename)
		if err != nil {
			return ui.Transition{}, err
		}
		switch command {
		case "open":
			if err := profile.Check(); err != nil {
				return ui.Transition{}, err
			}
			open := a.openWith
			if open == nil {
				open = platform.OpenWith
			}
			if err := open(profile.Path); err != nil {
				return ui.Transition{}, fault.Wrap("openFailed", err)
			}
			return a.done("openDialogShown", displayName(filename)), nil
		case "show":
			return ui.Transition{Result: ui.Result{Data: profile.Data}}, nil
		case "rename":
			return ui.Transition{Next: &ui.Step{Question: a.Language.Text("renameQuestion"), Hint: a.Language.Text("renameHint"), Initial: displayName(filename), Submit: func(name string) (ui.Transition, error) {
				if err := a.commit(commitRequest{Operation: "rename", Root: s.Root, Config: &state.Config, Source: &profile, Value: name}, func() error { return s.Rename(state, profile, name) }); err != nil {
					return ui.Transition{}, err
				}
				return a.done("renamed", name), nil
			}}}, nil
		case "remove":
			return ui.Transition{Next: a.confirm(a.Language.Text("removeQuestion", displayName(filename)), func(yes bool) (ui.Transition, error) {
				if !yes {
					return a.cancelled(), nil
				}
				if err := a.commit(commitRequest{Operation: "remove", Root: s.Root, Config: &state.Config, Source: &profile}, func() error { return s.Remove(state, profile) }); err != nil {
					return ui.Transition{}, err
				}
				return a.done("removed", displayName(filename)), nil
			})}, nil
		}
		return ui.Transition{}, fault.New("usage")
	}
	return step
}

func (a *App) importStep(s core.Store, state core.State) *ui.Step {
	completion := newPathCompleter(false)
	return &ui.Step{Question: a.Language.Text("sourceQuestion"), Help: a.Language.Text("pathHelp"), Complete: completion.Complete, Suggest: completion.Suggest,
		Submit: func(input string) (ui.Transition, error) {
			plan, err := s.PrepareImport(input)
			if err != nil {
				return ui.Transition{}, err
			}
			commit := func() (ui.Transition, error) {
				if err := a.commit(commitRequest{Operation: "import", Root: s.Root, Config: &state.Config, Source: &plan.Source, Target: &plan.Target, Value: plan.Filename}, func() error { return s.Import(state, plan) }); err != nil {
					return ui.Transition{}, err
				}
				result := a.done("imported", displayName(plan.Filename))
				cleanup := a.confirm(a.Language.Text("deleteSourceQuestion"), func(yes bool) (ui.Transition, error) {
					if yes {
						if err := a.commit(commitRequest{Operation: "deleteSource", Root: s.Root, Source: &plan.Source, TargetPath: plan.Target.Path}, plan.DeleteSource); err != nil {
							return ui.Transition{Result: ui.Result{Err: err}}, nil
						}
					}
					return result, nil
				})
				cleanup.Help = plan.Source.Path
				cleanup.CancelResult = &result.Result
				return ui.Transition{Next: cleanup}, nil
			}
			if plan.Target.Info == nil {
				return commit()
			}
			return ui.Transition{Next: a.confirm(a.Language.Text("overwriteQuestion", displayName(plan.Filename)), func(yes bool) (ui.Transition, error) {
				if !yes {
					return a.cancelled(), nil
				}
				return commit()
			})}, nil
		}}
}

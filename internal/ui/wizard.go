// Package ui implements one inline program for an entire command's wizard.
package ui

import (
	"errors"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/i18n"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
)

type Option struct {
	Label, Value string
	Current      bool
}

// Step describes interaction only; Submit belongs to the CLI business flow.
type Step struct {
	Question, Help, Initial string
	// Placeholder is shown for empty input and accepted with Enter or Tab.
	Placeholder string
	Hint        string
	Confirm     bool
	Options     []Option
	Selected    int
	Complete    func(string) (string, error)
	Suggest     func(string) string
	Submit      func(string) (Transition, error)
	// CancelResult applies when an earlier step already committed an operation.
	CancelResult *Result
}

type Result struct {
	Message string
	Data    []byte
	Code    int
	Err     error
}
type Transition struct {
	Next   *Step
	Result Result
}
type submitted struct {
	transition Transition
	err        error
	answer     string
}
type completion struct {
	value string
	err   error
}

type suggestionTick struct {
	version uint64
	input   string
}
type suggested struct {
	version uint64
	value   string
}

type Notice struct {
	Text    string
	Warning bool
}

type Model struct {
	step           *Step
	input          textinput.Model
	selected       int
	width, height  int
	busy, done     bool
	err            error
	completed      []string
	Result         Result
	lang           i18n.Language
	color          bool
	notice         Notice
	updates        <-chan Notice
	closed         <-chan struct{}
	suggestion     string
	suggestVersion uint64
}

func New(step *Step, lang i18n.Language, color bool) *Model {
	m := &Model{width: 80, height: 24, lang: lang, color: color}
	m.activate(step)
	return m
}

func (m *Model) activate(step *Step) {
	m.step = step
	m.selected = step.Selected
	m.input = textinput.New()
	m.input.Prompt = ": "
	m.input.SetVirtualCursor(false)
	m.input.SetStyles(textinput.Styles{})
	m.input.KeyMap.Paste.SetEnabled(false)
	m.input.SetValue(step.Initial)
	m.input.CursorEnd()
	m.input.SetWidth(max(1, m.width-4))
	m.input.Focus()
	m.err = nil
	m.suggestion = ""
	m.suggestVersion++
}

func (m *Model) Init() tea.Cmd { return tea.Batch(m.awaitNotice(), m.queueSuggestion()) }

// Debounce reads and reject stale results without changing the actual input.
func (m *Model) queueSuggestion() tea.Cmd {
	m.suggestion = ""
	m.suggestVersion++
	if m.done || m.busy || m.step.Suggest == nil || len(m.step.Options) > 0 || (m.input.Value() == "" && m.step.Placeholder != "") || m.input.Position() != len([]rune(m.input.Value())) {
		return nil
	}
	version, input := m.suggestVersion, m.input.Value()
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return suggestionTick{version, input} })
}

func (m *Model) awaitNotice() tea.Cmd {
	if m.updates == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case notice := <-m.updates:
			return notice
		case <-m.closed:
			return nil
		}
	}
}

func (m *Model) finish(result Result) (tea.Model, tea.Cmd) {
	m.done, m.Result = true, result
	m.input.Blur()
	return m, tea.Quit
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case suggestionTick:
		if msg.version != m.suggestVersion || m.done || m.busy {
			return m, nil
		}
		suggest := m.step.Suggest
		return m, func() tea.Msg { return suggested{msg.version, suggest(msg.input)} }
	case suggested:
		if msg.version == m.suggestVersion && !m.done && !m.busy && strings.HasPrefix(msg.value, m.input.Value()) {
			m.suggestion = msg.value
		}
		return m, nil
	case Notice:
		if m.busy {
			m.notice = msg
		}
		return m, m.awaitNotice()
	case tea.WindowSizeMsg:
		m.width, m.height = max(4, msg.Width), max(4, msg.Height)
		m.input.SetWidth(max(1, m.width-4))
		return m, nil
	case submitted:
		m.busy = false
		m.notice = Notice{}
		if msg.err != nil {
			if platform.PermissionDenied(msg.err) {
				return m.finish(Result{Err: msg.err})
			}
			m.err = msg.err
			var e *fault.Error
			if errors.As(msg.err, &e) && e.Stop {
				return m.finish(Result{Err: msg.err})
			}
			if errors.As(msg.err, &e) && e.ClearInput {
				m.input.SetValue("")
			}
			return m, m.queueSuggestion()
		}
		theme := Theme{Color: m.color}
		line := theme.Success("✓") + " " + theme.Muted(i18n.Safe(m.step.Question)) + " " + theme.Accent(i18n.Safe(msg.answer))
		line = ansi.Truncate(line, m.width-1, "…")
		m.completed = append(m.completed, line)
		print := tea.Println(line)
		if msg.transition.Next == nil {
			m.done, m.Result = true, msg.transition.Result
			return m, tea.Sequence(print, tea.Quit)
		}
		m.activate(msg.transition.Next)
		return m, tea.Batch(print, m.queueSuggestion())
	case completion:
		if platform.PermissionDenied(msg.err) {
			return m.finish(Result{Err: msg.err})
		}
		m.busy = false
		m.err = msg.err
		if msg.err == nil {
			m.input.SetValue(msg.value)
			m.input.CursorEnd()
		}
		return m, m.queueSuggestion()
	case tea.PasteMsg:
		if m.busy || m.done || len(m.step.Options) > 0 {
			return m, nil
		}
		if strings.ContainsAny(msg.Content, "\r\n\x00") {
			m.err = fault.New("onePath")
			return m, nil
		}
	case tea.KeyPressMsg:
		if m.busy || m.done {
			return m, nil
		}
		switch msg.String() {
		case "esc", "ctrl+c":
			if m.step.CancelResult != nil {
				return m.finish(*m.step.CancelResult)
			}
			return m.finish(Result{Code: 2, Message: m.lang.Text("cancelled")})
		case "enter":
			m.suggestion = ""
			m.suggestVersion++
			value, answer := m.input.Value(), m.input.Value()
			if len(m.step.Options) == 0 && value == "" && m.step.Placeholder != "" {
				value, answer = m.step.Placeholder, m.step.Placeholder
				m.input.SetValue(value)
				m.input.CursorEnd()
			}
			if len(m.step.Options) > 0 {
				value = m.step.Options[m.selected].Value
				answer = m.step.Options[m.selected].Label
			}
			submit := m.step.Submit
			m.busy, m.err = true, nil
			return m, func() tea.Msg { next, err := submit(value); return submitted{next, err, answer} }
		case "tab":
			if len(m.step.Options) == 0 && m.input.Value() == "" && m.step.Placeholder != "" {
				m.input.SetValue(m.step.Placeholder)
				m.input.CursorEnd()
				return m, m.queueSuggestion()
			}
			if m.step.Complete != nil {
				m.suggestion = ""
				m.suggestVersion++
				complete, value := m.step.Complete, m.input.Value()
				m.busy = true
				return m, func() tea.Msg { value, err := complete(value); return completion{value, err} }
			}
		case "up", "k":
			if len(m.step.Options) > 0 {
				m.selected = max(0, m.selected-1)
				return m, nil
			}
		case "down", "j":
			if len(m.step.Options) > 0 {
				m.selected = min(len(m.step.Options)-1, m.selected+1)
				return m, nil
			}
		}
	}
	if m.done || m.busy || len(m.step.Options) > 0 {
		return m, nil
	}
	var cmd tea.Cmd
	previous, position := m.input.Value(), m.input.Position()
	m.input, cmd = m.input.Update(msg)
	if previous != m.input.Value() || position != m.input.Position() {
		return m, tea.Batch(cmd, m.queueSuggestion())
	}
	return m, cmd
}

func (m *Model) View() tea.View {
	if m.done {
		return tea.NewView("")
	}
	theme := Theme{Color: m.color}
	prefix := ">"
	if m.step.Confirm || len(m.step.Options) > 0 {
		prefix = "?"
	}
	question := theme.Focus(prefix) + " " + theme.Heading(i18n.Safe(m.step.Question))
	lines := []string{question}
	cursorX, cursorY := 0, 1
	if m.step.Hint != "" {
		lines = append(lines, theme.Hint(i18n.Safe(m.step.Hint)))
		cursorY++
	}
	if len(m.step.Options) > 0 {
		hasCurrent := false
		for _, option := range m.step.Options {
			hasCurrent = hasCurrent || option.Current
		}
		visible := max(1, m.height-5)
		start := max(0, m.selected-visible+1)
		for i := start; i < min(len(m.step.Options), start+visible); i++ {
			option := m.step.Options[i]
			pointer, label := "  ", i18n.Safe(option.Label)
			if i == m.selected {
				pointer, label = theme.Focus("> "), theme.Focus(label)
			} else if option.Current {
				label = theme.Success(label)
			}
			mark := ""
			if hasCurrent {
				mark = "  "
				if option.Current {
					mark = theme.Success("✓") + " "
				}
			}
			lines = append(lines, pointer+mark+label)
		}
	} else {
		text, column := m.inputLine(max(1, m.width-4))
		lines = append(lines, theme.Accent(":")+" "+strings.TrimPrefix(text, ": "))
		cursorX = column
	}
	help := m.step.Help
	if help == "" {
		if len(m.step.Options) > 0 {
			help = m.lang.Text("chooseHelp")
		} else {
			help = m.lang.Text("inputHelp")
		}
	}
	if m.busy {
		if m.notice.Text != "" {
			if m.notice.Warning {
				lines = append(lines, theme.Feedback(i18n.Safe(m.notice.Text), true))
			} else {
				lines = append(lines, theme.Hint(i18n.Safe(m.notice.Text)))
			}
		} else {
			lines = append(lines, theme.Hint(m.lang.Text("working")))
		}
	} else {
		lines = append(lines, theme.Hint(i18n.Safe(help)))
	}
	if m.err != nil {
		lines = append(lines, theme.Feedback(m.lang.Error(m.err), fault.IsWarning(m.err)))
	}
	for i := range lines {
		lines[i] = ansi.Truncate(lines[i], m.width-1, "…")
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	if len(m.step.Options) == 0 && !m.busy {
		v.Cursor = tea.NewCursor(cursorX, cursorY)
		v.Cursor.Blink = false
	}
	return v
}

// Bubbles owns editing; this viewport measures the cursor in display columns.
func (m *Model) inputLine(available int) (string, int) {
	if m.input.Value() == "" && m.step.Placeholder != "" {
		placeholder := ansi.Truncate(i18n.Safe(m.step.Placeholder), available, "…")
		return ": " + (Theme{Color: m.color}).Muted(placeholder), 2
	}
	runes := []rune(m.input.Value())
	position := min(m.input.Position(), len(runes))
	before := i18n.Safe(string(runes[:position]))
	after := i18n.Safe(string(runes[position:]))
	before = ansi.TruncateLeft(before, max(0, ansi.StringWidth(before)-available), "")
	if position == len(runes) && !m.busy && strings.HasPrefix(m.suggestion, m.input.Value()) && len(m.suggestion) > len(m.input.Value()) {
		suffix := i18n.Safe(m.suggestion[len(m.input.Value()):])
		if !m.color {
			suffix = "[" + suffix + "]"
		}
		after = (Theme{Color: m.color}).Muted(suffix)
	}
	return ": " + before + ansi.Truncate(after, max(0, available-ansi.StringWidth(before)), ""), 2 + ansi.StringWidth(before)
}

func Run(step *Step, lang i18n.Language) Result {
	return RunWithStatus(step, lang, nil)
}

func RunWithStatus(step *Step, lang i18n.Language, updates <-chan Notice) Result {
	color := OutputTheme(os.Stderr).Color
	m := New(step, lang, color)
	closed := make(chan struct{})
	defer close(closed)
	m.updates, m.closed = updates, closed
	profile := colorprofile.ANSI
	if !color {
		profile = colorprofile.ASCII
	}
	model, err := tea.NewProgram(m, tea.WithInput(os.Stdin), tea.WithOutput(os.Stderr), tea.WithColorProfile(profile)).Run()
	if err != nil {
		return Result{Code: 1, Err: err}
	}
	return model.(*Model).Result
}

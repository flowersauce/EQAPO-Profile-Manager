package ui

import (
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/fault"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/i18n"
)

func key(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func TestCompletedSnapshotIsNotPartOfDynamicView(t *testing.T) {
	second := &Step{Question: "Second", Submit: func(string) (Transition, error) { return Transition{}, nil }}
	m := New(&Step{Question: "First", Initial: "answer", Submit: func(string) (Transition, error) { return Transition{Next: second}, nil }}, i18n.ForLocale("en"), false)
	_, cmd := m.Update(key(tea.KeyEnter))
	if cmd == nil {
		t.Fatal("submission command missing")
	}
	m.Update(cmd())
	if len(m.completed) != 1 || m.completed[0] != "✓ First answer" {
		t.Fatalf("snapshot: %v", m.completed)
	}
	if strings.Contains(m.View().Content, "First") || !strings.Contains(m.View().Content, "Second") {
		t.Fatal("completed step remains in erasable view")
	}
	m.Update(key(tea.KeyEscape))
	if m.View().Content != "" || m.Result.Code != 2 || len(m.completed) != 1 {
		t.Fatal("cancel did not clean current view")
	}
}

func TestUnhandledPermissionFailureClosesWizard(t *testing.T) {
	m := New(&Step{Question: "Import"}, i18n.ForLocale("en"), false)
	m.Update(submitted{err: os.ErrPermission})
	if !m.done || m.Result.Err == nil || m.View().Content != "" {
		t.Fatal("permission error left an active terminal UI")
	}
}

func TestValidationRetainsInputExceptRenameCollision(t *testing.T) {
	m := New(&Step{Question: "Name", Initial: "original"}, i18n.ForLocale("en"), false)
	m.Update(submitted{err: fault.New("filename")})
	if m.input.Value() != "original" || m.err == nil {
		t.Fatal("validation lost input")
	}
	e := fault.New("collision")
	e.ClearInput = true
	m.Update(submitted{err: e})
	if m.input.Value() != "" || m.err == nil {
		t.Fatal("collision did not clear input")
	}
}

func TestChoicesAndCommittedCancellation(t *testing.T) {
	result := Result{Message: "Imported", Code: 0}
	m := New(&Step{Question: "Delete source?", Selected: 1, Options: []Option{{Label: "Yes", Value: "yes"}, {Label: "No", Value: "no"}}, CancelResult: &result}, i18n.ForLocale("en"), false)
	m.Update(key('y'))
	if m.selected != 1 {
		t.Fatal("hidden y shortcut accepted")
	}
	m.Update(key('k'))
	if m.selected != 0 {
		t.Fatal("k did not move")
	}
	m.Update(key(tea.KeyEscape))
	if m.Result.Code != 0 || m.Result.Message != "Imported" {
		t.Fatal("cancel undid committed import")
	}
}

func TestUnicodeViewportAndPlainRendering(t *testing.T) {
	value := `C:\非常长的中文目录\耳机配置.txt`
	m := New(&Step{Question: "文件路径", Initial: value}, i18n.ForLocale("zh-CN"), false)
	for _, width := range []int{80, 15, 7} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		view := m.View()
		if strings.Contains(view.Content, "\x1b[") {
			t.Fatal("plain view contains ANSI")
		}
		for _, line := range strings.Split(view.Content, "\n") {
			if ansi.StringWidth(line) >= width {
				t.Fatalf("line wraps at %d: %q", width, line)
			}
		}
		if view.Cursor == nil || view.Cursor.X >= width || view.Cursor.Y != 1 || m.input.Value() != value {
			t.Fatal("cursor or value corrupted")
		}
		if !strings.HasPrefix(strings.Split(view.Content, "\n")[1], ": ") {
			t.Fatal("input must remain on a separate colon-prefixed line")
		}
	}
	m.Update(tea.PasteMsg{Content: "one\ntwo"})
	if m.err == nil || m.input.Value() != value {
		t.Fatal("multiline paste was merged into one path")
	}
}

func TestPromptKindsAndHintCursorPosition(t *testing.T) {
	lang := i18n.ForLocale("zh-CN")
	m := New(&Step{Question: lang.Text("renameQuestion"), Hint: lang.Text("renameHint"), Initial: "Philips SHP9500 GraphicEq"}, lang, false)
	want := "> 输入新文件名称\n^ 无需输入文件后缀\n: Philips SHP9500 GraphicEq\n"
	if !strings.HasPrefix(m.View().Content, want) || m.View().Cursor == nil || m.View().Cursor.Y != 2 {
		t.Fatalf("hint displaced input/cursor: %+v", m.View())
	}
	m.activate(&Step{Question: "Confirm?", Confirm: true, Options: []Option{{Label: "No"}}})
	if !strings.HasPrefix(m.View().Content, "? Confirm?\n") {
		t.Fatal("confirmation marker missing")
	}
	m.activate(&Step{Question: "Select", Options: []Option{{Label: "Profile"}}})
	if !strings.HasPrefix(m.View().Content, "? Select\n") {
		t.Fatal("selection marker missing")
	}
}

func TestUACNoticeAndRefusalKeepCurrentInput(t *testing.T) {
	step := &Step{Question: "Filename", Initial: "chosen name"}
	m := New(step, i18n.ForLocale("en"), false)
	m.busy = true
	m.Update(Notice{Text: "Approve UAC", Warning: true})
	if !strings.Contains(m.View().Content, "! Approve UAC") {
		t.Fatal("UAC warning has no marker")
	}
	m.Update(submitted{err: fault.New("elevationCancelled")})
	if m.done || m.step != step || m.input.Value() != "chosen name" || !strings.Contains(m.View().Content, "! Administrator access was declined") {
		t.Fatal("UAC refusal restarted or cleared the original step")
	}
	m.Update(submitted{err: fault.New("filename")})
	if !strings.Contains(m.View().Content, "✗ ") {
		t.Fatal("error uses a warning marker")
	}
}

func TestSuggestionIsDisplayOnlyAndStaleResultsAreIgnored(t *testing.T) {
	input := `C:\Ear`
	submittedValue := ""
	step := &Step{Question: "Path", Initial: input, Suggest: func(string) string { return `C:\Earphones\` }, Submit: func(value string) (Transition, error) {
		submittedValue = value
		return Transition{}, nil
	}}
	m := New(step, i18n.ForLocale("en"), false)
	m.queueSuggestion()
	version := m.suggestVersion
	m.Update(suggested{version: version, value: `C:\Earphones\`})
	if m.input.Value() != input || !strings.Contains(m.View().Content, `: C:\Ear[phones\]`) {
		t.Fatal("preview changed input or was not displayed")
	}
	m.Update(key(tea.KeyLeft))
	m.Update(suggested{version: version, value: `C:\Earphones\`})
	if m.suggestion != "" {
		t.Fatal("stale suggestion survived cursor movement")
	}
	_, submit := m.Update(key(tea.KeyEnter))
	submit()
	if submittedValue != input {
		t.Fatal("Enter submitted unaccepted suggestion")
	}
}

func TestPlaceholderStaysEmptyUntilAccepted(t *testing.T) {
	defaultPath := `C:\Program Files\EqualizerAPO`
	var accepted string
	m := New(&Step{Question: "Path", Placeholder: defaultPath, Suggest: func(string) string { return "unexpected" }, Submit: func(value string) (Transition, error) {
		accepted = value
		return Transition{}, nil
	}}, i18n.ForLocale("en"), false)
	if m.input.Value() != "" || !strings.Contains(m.View().Content, ": "+defaultPath) || m.View().Cursor.X != 2 {
		t.Fatal("placeholder was prefilled as input")
	}
	if m.queueSuggestion() != nil {
		t.Fatal("completion preview competed with placeholder")
	}
	m.Update(key('x'))
	if m.input.Value() != "x" || strings.Contains(m.View().Content, defaultPath) {
		t.Fatal("typing appended to the default")
	}
	m.Update(key(tea.KeyBackspace))
	if m.input.Value() != "" || !strings.Contains(m.View().Content, defaultPath) {
		t.Fatal("clearing did not restore placeholder")
	}
	_, cmd := m.Update(key(tea.KeyEnter))
	result := cmd().(submitted)
	if accepted != defaultPath || result.answer != defaultPath {
		t.Fatal("Enter did not submit and record the default")
	}
}

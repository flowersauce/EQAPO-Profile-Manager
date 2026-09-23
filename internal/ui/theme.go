package ui

import (
	"io"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/flowersauce/EQAPO-Profile-Manager/internal/platform"
)

// Theme shares semantic ANSI colors between the wizard and static CLI output.
type Theme struct{ Color bool }

func OutputTheme(output io.Writer) Theme {
	_, noColor := os.LookupEnv("NO_COLOR")
	file, terminal := output.(*os.File)
	return Theme{Color: !noColor && terminal && platform.IsTerminal(file) && platform.IsTerminal(os.Stdout)}
}

func (t Theme) style(text, color string, bold bool) string {
	if !t.Color {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Bold(bold).Render(text)
}

func (t Theme) Focus(text string) string   { return t.style(text, "6", true) }
func (t Theme) Link(text string) string    { return t.style(text, "4", false) }
func (t Theme) Accent(text string) string  { return t.style(text, "5", false) }
func (t Theme) Success(text string) string { return t.style(text, "2", false) }
func (t Theme) Warning(text string) string { return t.style(text, "3", false) }
func (t Theme) Muted(text string) string   { return t.style(text, "8", false) }
func (t Theme) Error(text string) string   { return t.style(text, "1", false) }
func (t Theme) Hint(text string) string    { return t.Muted("^ " + text) }
func (t Theme) Feedback(text string, warning bool) string {
	if warning {
		return t.Warning("! " + text)
	}
	return t.Error("✗ " + text)
}
func (t Theme) Heading(text string) string {
	if !t.Color {
		return text
	}
	return lipgloss.NewStyle().Bold(true).Render(text)
}

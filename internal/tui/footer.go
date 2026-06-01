package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type footerBinding struct {
	keys string
	desc string
}

var (
	footerKeyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	footerDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	footerStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func formFooterBindings(f formModel) []footerBinding {
	if f.currentStep() == StepSummary {
		return []footerBinding{
			{"s", "save"},
			{"e", "edit"},
			{"esc", "cancel"},
		}
	}
	bindings := []footerBinding{
		{"enter", "continue"},
		{"esc", "cancel"},
		{"ctrl+←", "prev step"},
		{"ctrl+→", "next step"},
	}
	if f.isChoiceStep() {
		bindings = append([]footerBinding{
			{"↑/k", "up"},
			{"↓/j", "down"},
		}, bindings...)
	}
	return bindings
}

func renderBinding(b footerBinding) string {
	return footerKeyStyle.Render(b.keys) + " " + footerDescStyle.Render(b.desc)
}

func renderFooterHelp(bindings []footerBinding) string {
	parts := make([]string, len(bindings))
	for i, b := range bindings {
		parts[i] = renderBinding(b)
	}
	return strings.Join(parts, " • ")
}

// renderFooterBar is a single-line key hint row plus the bottom rule.
func renderFooterBar(w int, bindings []footerBinding) string {
	help := renderFooterHelp(bindings)
	return strings.Join([]string{
		footerStyle.Width(w).Align(lipgloss.Center).Render(help),
		horizontalRule(w),
	}, "\n")
}

func renderBottomBar(w int, bindings []footerBinding, status string) string {
	var lines []string
	if status != "" {
		lines = append(lines, footerStatusStyle.Width(w).Align(lipgloss.Center).Render(status))
	}
	lines = append(lines, footerStyle.Width(w).Align(lipgloss.Center).Render(renderFooterHelp(bindings)))
	lines = append(lines, horizontalRule(w))
	return strings.Join(lines, "\n")
}

func formBottomBar(w int, f formModel) string {
	return renderFooterBar(w, formFooterBindings(f))
}

func confirmBottomBar(w int, alias string, defaultYes bool) string {
	status := fmt.Sprintf("Delete profile %q? %s", alias, confirmYNPrompt(defaultYes))
	return renderBottomBar(w, confirmFooterBindings(defaultYes), status)
}

func quitConfirmBottomBar(w int) string {
	status := "Quit mewsh? " + confirmYNPrompt(true)
	return renderBottomBar(w, confirmFooterBindings(true), status)
}

func killConfirmBottomBar(w int, prompt string, defaultYes bool) string {
	return renderBottomBar(w, confirmFooterBindings(defaultYes), prompt)
}

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

func listFooterBindings() []footerBinding {
	return []footerBinding{
		{"↑/k", "up"},
		{"↓/j", "down"},
		{"/", "filter"},
		{"enter", "connect"},
		{"m", "menu"},
		{"esc/q", "quit"},
		{"?", "more"},
	}
}

func listMenuFooterBindings() []footerBinding {
	return []footerBinding{
		{"a", "add"},
		{"e", "edit"},
		{"d", "delete"},
		{"s", "sessions"},
		{"m", "back"},
	}
}

func formFooterBindings() []footerBinding {
	return []footerBinding{
		{"enter", "next"},
		{"esc", "cancel"},
		{"tab", "next field"},
		{"shift+tab", "prev field"},
		{"↑/k", "up"},
		{"↓/j", "down"},
	}
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

func renderBottomBar(w int, bindings []footerBinding, status string) string {
	var lines []string
	if status != "" {
		lines = append(lines, footerStatusStyle.Width(w).Align(lipgloss.Center).Render(status))
	}
	help := renderFooterHelp(bindings)
	lines = append(lines, footerStyle.Width(w).Align(lipgloss.Center).Render(help))
	lines = append(lines, horizontalRule(w))
	return strings.Join(lines, "\n")
}

func (m listModel) footerStatus(profileCount int, activeSessions int, extra string) string {
	if extra != "" {
		return extra
	}
	if activeSessions > 0 {
		return fmt.Sprintf("%d profiles | Active sessions: %d", profileCount, activeSessions)
	}
	return fmt.Sprintf("%d profiles", profileCount)
}

func listBottomBar(w int, m listModel, profileCount int, menuOpen bool, activeSessions int, statusOverride string) string {
	bindings := listFooterBindings()
	if menuOpen {
		bindings = listMenuFooterBindings()
	}
	return renderBottomBar(w, bindings, m.footerStatus(profileCount, activeSessions, statusOverride))
}

func formBottomBar(w int, step, total int) string {
	status := ""
	if total > 0 {
		status = fmt.Sprintf("Step %d/%d", step, total)
	}
	return renderBottomBar(w, formFooterBindings(), status)
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

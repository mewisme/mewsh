package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	helpKeysCol      = 18
	helpPanelPadH    = 4 // left+right padding inside border
	helpPanelBorderW = 2
)

type helpBinding struct {
	keys string
	desc string
}

type helpPage struct {
	title string
	about string
	items []helpBinding
}

func helpPages() []helpPage {
	return []helpPage{
		{
			title: "Global",
			about: "Shortcuts that work from almost anywhere in the TUI.",
			items: []helpBinding{
				{"?", "Open or close this help guide"},
				{"←/h  →/l", "Previous / next help page"},
				{"esc", "Close help, or go back one screen"},
				{"q", "Quit mewsh (asks for confirmation)"},
				{"ctrl+c", "Quit mewsh (asks for confirmation)"},
			},
		},
		{
			title: "Profiles",
			about: "Main screen: browse saved SSH profiles, filter the list, and connect.",
			items: []helpBinding{
				{"↑/k  ↓/j", "Move selection up or down"},
				{"/", "Filter profiles by alias, host, user, or note"},
				{"enter", "Connect to the selected profile (opens SSH in a new terminal)"},
				{"m", "Open the profile menu (add, edit, delete, sessions)"},
				{"?", "Open this help guide"},
				{"esc esc", "Press Esc twice quickly to quit"},
				{"q", "Quit mewsh (confirmation prompt)"},
			},
		},
		{
			title: "Menu",
			about: "Opened from the profile list with m. Manage profiles and view active sessions.",
			items: []helpBinding{
				{"a", "Add a new profile (step-by-step form)"},
				{"e", "Edit the selected profile"},
				{"d", "Delete the selected profile (confirmation)"},
				{"s", "View and manage active SSH sessions"},
				{"m", "Close the menu and return to the profile list"},
			},
		},
		{
			title: "Sessions",
			about: "Lists SSH sessions started by mewsh. Mark one or many, then kill.",
			items: []helpBinding{
				{"↑/k  ↓/j", "Move selection up or down"},
				{"/", "Filter sessions by alias or target"},
				{"space", "Toggle mark on the selected session"},
				{"enter", "Kill marked session(s), or the selected one if none marked"},
				{"a", "Kill all listed sessions (confirmation)"},
				{"m  esc", "Return to the profile list"},
				{"?", "Open this help guide"},
			},
		},
		{
			title: "Add / edit profile",
			about: "Multi-step form for creating or updating a profile.",
			items: []helpBinding{
				{"enter", "Confirm the current field and go to the next step"},
				{"esc", "Cancel and discard changes"},
				{"tab", "Move to the next field on this step"},
				{"shift+tab", "Move to the previous field on this step"},
				{"↑/k  ↓/j", "Change choice on connection type or auth type steps"},
			},
		},
		{
			title: "Confirmations",
			about: "Yes/no prompts for delete, quit, kill session, and similar actions.",
			items: []helpBinding{
				{"Y/enter", "Yes (default on quit)"},
				{"N/enter", "No (default on delete / kill)"},
				{"y / n", "Yes / no"},
				{"esc", "Cancel (same as no)"},
			},
		},
	}
}

var (
	helpTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	helpAboutStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginBottom(1)
	helpKeysStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpDescStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	helpPanelStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("69")).Padding(1, 2)
	helpPageNumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	helpNavStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func helpPanelWidth(termW int, page helpPage, pageIdx, pageCount int) int {
	footer := fmt.Sprintf("Page %d of %d", pageIdx+1, pageCount)
	nav := "←/h prev   →/l next   ? or esc close"

	maxContent := lipgloss.Width(page.title)
	maxContent = max(maxContent, lipgloss.Width(page.about))
	maxContent = max(maxContent, lipgloss.Width(footer))
	maxContent = max(maxContent, lipgloss.Width(nav))
	for _, item := range page.items {
		maxContent = max(maxContent, helpKeysCol+2+lipgloss.Width(item.desc))
	}

	panel := maxContent + helpPanelPadH + helpPanelBorderW
	maxPanel := termW - 2
	if maxPanel < 40 {
		maxPanel = 40
	}
	if panel > maxPanel {
		panel = maxPanel
	}
	if panel < 44 {
		panel = min(44, maxPanel)
	}
	return panel
}

func padHelpKeys(keys string) string {
	rendered := helpKeysStyle.Render(keys)
	w := lipgloss.Width(rendered)
	if w >= helpKeysCol {
		return rendered
	}
	return rendered + strings.Repeat(" ", helpKeysCol-w)
}

func wrapWords(text string, width int) []string {
	if width < 8 {
		width = 8
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	var line strings.Builder
	for _, word := range words {
		add := word
		if line.Len() == 0 {
			line.WriteString(add)
			continue
		}
		candidate := line.String() + " " + add
		if lipgloss.Width(candidate) <= width {
			line.Reset()
			line.WriteString(candidate)
			continue
		}
		lines = append(lines, line.String())
		line.Reset()
		line.WriteString(add)
	}
	if line.Len() > 0 {
		lines = append(lines, line.String())
	}
	return lines
}

func renderHelpRow(keys, desc string, descW int) string {
	keysPart := padHelpKeys(keys)
	if lipgloss.Width(desc) <= descW {
		return keysPart + helpDescStyle.Render(desc)
	}
	lines := wrapWords(desc, descW)
	if len(lines) == 0 {
		return keysPart
	}
	var b strings.Builder
	b.WriteString(keysPart)
	b.WriteString(helpDescStyle.Render(lines[0]))
	if len(lines) == 1 {
		return b.String()
	}
	indent := strings.Repeat(" ", helpKeysCol)
	for _, ln := range lines[1:] {
		b.WriteString("\n")
		b.WriteString(indent)
		b.WriteString(helpDescStyle.Render(ln))
	}
	return b.String()
}

func renderHelpPage(page helpPage, pageIdx, pageCount, termW int) string {
	panelW := helpPanelWidth(termW, page, pageIdx, pageCount)
	contentW := panelW - helpPanelPadH - helpPanelBorderW
	descW := contentW - helpKeysCol
	if descW < 16 {
		descW = 16
	}

	var b strings.Builder
	b.WriteString(helpTitleStyle.Render(page.title))
	b.WriteString("\n")

	aboutLines := wrapWords(page.about, contentW)
	if len(aboutLines) == 0 {
		b.WriteString(helpAboutStyle.Render(page.about))
	} else {
		b.WriteString(helpAboutStyle.Render(aboutLines[0]))
		for _, ln := range aboutLines[1:] {
			b.WriteString("\n")
			b.WriteString(helpAboutStyle.Render(ln))
		}
	}
	b.WriteString("\n")

	for _, item := range page.items {
		b.WriteString(renderHelpRow(item.keys, item.desc, descW))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	footer := fmt.Sprintf("Page %d of %d", pageIdx+1, pageCount)
	b.WriteString(helpPageNumStyle.Render(footer))
	b.WriteString("\n")
	b.WriteString(helpNavStyle.Render("←/h prev   →/l next   ? or esc close"))

	return helpPanelStyle.Width(panelW).Render(strings.TrimSuffix(b.String(), "\n"))
}

func layoutHelpScreen(w, h, pageIdx int) string {
	w, h = clampDims(w, h)
	pages := helpPages()
	if pageIdx < 0 {
		pageIdx = 0
	}
	if pageIdx >= len(pages) {
		pageIdx = len(pages) - 1
	}

	header := renderHeader(w)
	content := renderHelpPage(pages[pageIdx], pageIdx, len(pages), w)
	bodyH := max(4, h-lipgloss.Height(header)-1)
	body := lipgloss.Place(w, bodyH, lipgloss.Center, lipgloss.Center, content)
	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func (m *Model) helpPageCount() int {
	return len(helpPages())
}

func (m *Model) helpPrev() {
	n := m.helpPageCount()
	if n == 0 {
		return
	}
	m.helpPage = (m.helpPage - 1 + n) % n
}

func (m *Model) helpNext() {
	n := m.helpPageCount()
	if n == 0 {
		return
	}
	m.helpPage = (m.helpPage + 1) % n
}

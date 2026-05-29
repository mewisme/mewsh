package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mewisme/mewsh/internal/connect"
)

const (
	defaultTermWidth  = 80
	defaultTermHeight = 24
	headerTitle       = "mewsh"
	headerSubtitle    = "Manage SSH credentials and connect servers."
)

var (
	subtitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginBottom(1)
	ruleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func horizontalRule(w int) string {
	if w < 1 {
		w = defaultTermWidth
	}
	return ruleStyle.Render(strings.Repeat("─", w))
}

// centerColumn places a block that is already contentColumnWidth(screenW) wide
// into the horizontal center of the terminal.
func centerColumn(screenW int, block string) string {
	if screenW < 1 {
		screenW = defaultTermWidth
	}
	return lipgloss.PlaceHorizontal(screenW, lipgloss.Center, block)
}

func renderHeader(screenW int) string {
	cw := listContentWidth(screenW)
	title := titleStyle.Width(cw).Align(lipgloss.Center).Render(headerTitle)
	subtitle := subtitleStyle.Width(cw).Align(lipgloss.Center).Render(headerSubtitle)
	inner := lipgloss.JoinVertical(lipgloss.Left, horizontalRule(cw), title, subtitle)
	return centerColumn(screenW, inner)
}

func clampDims(w, h int) (int, int) {
	if w < 1 {
		w = defaultTermWidth
	}
	if h < 1 {
		h = defaultTermHeight
	}
	return w, h
}

func headerHeight(w int) int {
	return lipgloss.Height(renderHeader(w))
}

func bottomBarHeight(w int, bottom string) int {
	if bottom == "" {
		return 0
	}
	return lipgloss.Height(bottom)
}

func errLineHeight(errMsg string) int {
	if errMsg == "" {
		return 0
	}
	return lipgloss.Height(errStyle.Render(errMsg))
}

// layoutScreen places the header at the top, body in the middle, and bottom bar
// (dim status + centered key help + rule) anchored at the terminal bottom.
func layoutScreen(w, h int, body, bottomBar, errMsg string) string {
	w, h = clampDims(w, h)
	header := renderHeader(w)
	hdrH := lipgloss.Height(header)
	botH := bottomBarHeight(w, bottomBar)
	errH := 0
	errLine := ""
	if errMsg != "" {
		errLine = errStyle.Width(w).Align(lipgloss.Center).Render(errMsg)
		errH = lipgloss.Height(errLine)
	}

	bodyH := max(4, h-hdrH-botH-errH)
	bodyBlock := lipgloss.Place(w, bodyH, lipgloss.Center, lipgloss.Center, centerColumn(w, body))

	parts := []string{header, bodyBlock}
	if errLine != "" {
		parts = append(parts, errLine)
	}
	if bottomBar != "" {
		parts = append(parts, bottomBar)
	}
	return strings.Join(parts, "\n")
}

func (m *Model) listBottomBar() string {
	w, _ := clampDims(m.width, m.height)
	status := ""
	if m.connecting {
		if m.connectingAlias != "" {
			status = fmt.Sprintf("Connecting to %s…", m.connectingAlias)
		} else {
			status = "Connecting…"
		}
	} else if m.quitEscPending {
		status = "Press esc again to quit"
	}
	return listBottomBar(w, m.list, len(m.cfg.Profiles), m.menuOpen, connect.ActiveSessionCount(), status)
}

func listContentWidth(screenW int) int {
	inner := screenW - 4
	if inner > 72 {
		inner = 72
	}
	if inner < 40 {
		inner = max(40, screenW-4)
	}
	return inner
}

func (m *Model) syncListSize() {
	w, h := clampDims(m.width, m.height)
	bottom := m.listBottomBar()
	used := headerHeight(w) + bottomBarHeight(w, bottom)
	if m.err != nil {
		used += errLineHeight(m.err.Error())
	}
	listH := max(4, h-used)
	m.list = m.list.withSize(listContentWidth(w), listH)
}

func (m *Model) sessionsBottomBar() string {
	w, _ := clampDims(m.width, m.height)
	return sessionsBottomBar(w, m.sessions)
}

func (m *Model) syncSessionsSize() {
	w, h := clampDims(m.width, m.height)
	bottom := m.sessionsBottomBar()
	used := headerHeight(w) + bottomBarHeight(w, bottom)
	if m.err != nil {
		used += errLineHeight(m.err.Error())
	}
	listH := max(4, h-used)
	m.sessions = m.sessions.withSize(listContentWidth(w), listH)
}

func (f *formModel) inputWidth() int {
	return max(10, f.boxWidth()-6)
}

func (f *formModel) boxWidth() int {
	w, _ := clampDims(f.width, f.height)
	inner := w - 4
	if inner > 72 {
		inner = 72
	}
	if inner < 40 {
		inner = max(40, w-4)
	}
	return inner
}

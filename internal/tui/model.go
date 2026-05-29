package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mewisme/mewsh/internal/config"
	"github.com/mewisme/mewsh/internal/connect"
	"github.com/mewisme/mewsh/internal/profile"
	"github.com/mewisme/mewsh/internal/secret"
	"github.com/mewisme/mewsh/internal/terminal"
)

type screen int

const (
	screenList screen = iota
	screenForm
	screenConfirmDelete
	screenConfirmKillSessions
	screenSessions
)

type Model struct {
	cfg               *config.Config
	screen            screen
	list              listModel
	sessions          sessionsListModel
	form              formModel
	confirmAlias      string
	confirmKillPrompt string
	confirmKillIDs    []string
	confirmKillAll    bool
	err               error
	connecting        bool
	connectingAlias   string
	menuOpen          bool
	quitting          bool
	width             int
	height            int
}

func NewModel(cfg *config.Config) Model {
	if cfg == nil {
		cfg = &config.Config{Profiles: []profile.Profile{}}
	}
	return Model{
		cfg:    cfg,
		screen: screenList,
		list:   newListModel(cfg.Profiles),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tea.WindowSize(), tea.SetWindowTitle(terminal.WindowTitle), sessionTickCmd())
}

func sessionTickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return sessionSyncMsg{} })
}

type sessionSyncMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		switch m.screen {
		case screenForm:
			m.form.applyWindowSize(msg.Width, msg.Height)
			return m, nil
		case screenList:
			m.syncListSize()
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		case screenSessions:
			m.syncSessionsSize()
			var cmd tea.Cmd
			m.sessions, cmd = m.sessions.Update(msg)
			return m, cmd
		default:
			return m, nil
		}
	case tea.KeyMsg:
		if m.screen == screenSessions && !m.sessions.filterInputActive() {
			switch msg.String() {
			case "esc", "m":
				m.screen = screenList
				m.syncListSize()
				return m, m.refreshLayout()
			}
		}
		if m.screen == screenList && !m.list.filterInputActive() {
			switch {
			case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "q", "esc"))):
				m.connecting = false
				connect.CleanupActive()
				m.quitting = true
				return m, tea.Quit
			}
		}
	case sessionSyncMsg:
		if m.quitting {
			return m, nil
		}
		connect.PruneStaleSessions()
		if m.screen == screenSessions {
			m.sessions = m.sessions.refresh()
		}
		return m, sessionTickCmd()
	case connectDoneMsg:
		m.connecting = false
		m.connectingAlias = ""
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
		}
		m.syncListSize()
		// Re-sync layout after a detached spawn (Windows ConPTY can desync until resized).
		return m, tea.Batch(sessionTickCmd(), m.refreshLayout(), tea.ClearScreen)
	case saveDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.cfg = msg.cfg
		m.list = newListModel(m.cfg.Profiles)
		m.syncListSize()
		m.screen = screenList
		return m, nil
	}

	switch m.screen {
	case screenList:
		return m.updateList(msg)
	case screenForm:
		return m.updateForm(msg)
	case screenConfirmDelete:
		return m.updateConfirmDelete(msg)
	case screenConfirmKillSessions:
		return m.updateConfirmKillSessions(msg)
	case screenSessions:
		return m.updateSessions(msg)
	}
	return m, nil
}

func (m Model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	// While typing a filter (including after opening the menu first), let the list
	// consume all keys so a/e/d/m/enter go into the filter — not menu actions.
	if _, ok := msg.(tea.KeyMsg); ok && m.list.filterInputActive() {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "m":
			m.menuOpen = !m.menuOpen
			m.syncListSize()
			return m, m.refreshLayout()
		case "a":
			if m.menuOpen {
				m.menuOpen = false
				m.form = newFormModel(nil, m.cfg)
				m.form.applyWindowSize(m.width, m.height)
				m.screen = screenForm
				m.err = nil
				return m, m.form.Init()
			}
		case "e":
			if m.menuOpen {
				if p := m.list.selectedProfile(); p != nil {
					m.menuOpen = false
					m.form = newFormModel(p, m.cfg)
					m.form.applyWindowSize(m.width, m.height)
					m.screen = screenForm
					m.err = nil
					return m, m.form.Init()
				}
			}
		case "d":
			if m.menuOpen {
				if p := m.list.selectedProfile(); p != nil {
					m.menuOpen = false
					m.confirmAlias = p.Alias
					m.screen = screenConfirmDelete
				}
				return m, nil
			}
		case "s":
			if m.menuOpen {
				m.menuOpen = false
				m.sessions = newSessionsListModel()
				m.screen = screenSessions
				m.err = nil
				m.syncSessionsSize()
				return m, m.refreshLayout()
			}
		case "enter":
			if m.connecting {
				return m, nil
			}
			if p := m.list.selectedProfile(); p != nil {
				m.err = nil
				m.connecting = true
				m.connectingAlias = p.Alias
				return m, tea.Batch(m.connectCmd(p.Alias), m.refreshLayout())
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	if m.form.cancelled {
		m.screen = screenList
		m.form.cancelled = false
		m.syncListSize()
		return m, m.refreshLayout()
	}
	if m.form.submitted {
		p := m.form.profile
		cfg := *m.cfg
		if m.form.editing {
			if idx := findIndex(cfg.Profiles, m.form.origAlias); idx >= 0 {
				cfg.Profiles[idx] = p
			}
		} else {
			cfg.Profiles = append(cfg.Profiles, p)
		}
		if m.form.password != "" {
			ref := p.PasswordRef
			if ref == "" {
				ref = p.Alias
			}
			if err := secret.SetPassword(ref, m.form.password); err != nil {
				m.err = err
				m.form.submitted = false
				m.form.stepIndex = len(m.form.steps) - 1
				m.form.syncStepInput()
				return m, nil
			}
		}
		if err := config.Save(&cfg); err != nil {
			m.err = err
			m.form.submitted = false
			m.form.stepIndex = len(m.form.steps) - 1
			m.form.syncStepInput()
			return m, nil
		}
		m.err = nil
		m.cfg = &cfg
		m.list = newListModel(cfg.Profiles)
		m.syncListSize()
		m.screen = screenList
		m.form.submitted = false
		return m, m.refreshLayout()
	}
	return m, cmd
}

func (m Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y":
			cfg := *m.cfg
			idx := findIndex(cfg.Profiles, m.confirmAlias)
			if idx >= 0 {
				ref := cfg.Profiles[idx].PasswordRef
				if ref == "" {
					ref = m.confirmAlias
				}
				cfg.Profiles = append(cfg.Profiles[:idx], cfg.Profiles[idx+1:]...)
				_ = secret.DeletePassword(ref)
				_ = config.Save(&cfg)
				m.cfg = &cfg
				m.list = newListModel(cfg.Profiles)
				m.syncListSize()
			}
			m.screen = screenList
			return m, m.refreshLayout()
		case "n", "N", "esc":
			m.screen = screenList
			return m, m.refreshLayout()
		}
	}
	return m, nil
}

func (m Model) updateConfirmKillSessions(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y":
			if m.confirmKillAll {
				connect.KillAllSessions()
				m.sessions.marked = map[string]bool{}
			} else {
				connect.KillSessions(m.confirmKillIDs)
				for _, id := range m.confirmKillIDs {
					delete(m.sessions.marked, id)
				}
			}
			m.confirmKillAll = false
			m.confirmKillIDs = nil
			m.confirmKillPrompt = ""
			m.sessions = m.sessions.refresh()
			m.syncSessionsSize()
			m.screen = screenSessions
			return m, m.refreshLayout()
		case "n", "N", "esc":
			m.confirmKillAll = false
			m.confirmKillIDs = nil
			m.confirmKillPrompt = ""
			m.screen = screenSessions
			return m, m.refreshLayout()
		}
	}
	return m, nil
}

func killSessionsConfirmPrompt(s sessionsListModel, ids []string) string {
	if len(ids) == 1 {
		for _, it := range s.items {
			if it.info.ID == ids[0] {
				return fmt.Sprintf("Kill session %q? (y/n)", it.info.Alias)
			}
		}
	}
	return fmt.Sprintf("Kill %d session(s)? (y/n)", len(ids))
}

func (m Model) updateSessions(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok && m.sessions.filterInputActive() {
		var cmd tea.Cmd
		m.sessions, cmd = m.sessions.Update(msg)
		return m, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case " ":
			m.sessions = m.sessions.toggleMark()
			return m, nil
		case "a", "A":
			if len(m.sessions.items) == 0 {
				return m, nil
			}
			m.confirmKillAll = true
			m.confirmKillIDs = nil
			m.confirmKillPrompt = fmt.Sprintf("Kill all %d session(s)? (y/n)", len(m.sessions.items))
			m.screen = screenConfirmKillSessions
			return m, nil
		case "enter":
			ids := m.sessions.markedIDs()
			if len(ids) == 0 {
				if it := m.sessions.selectedItem(); it != nil {
					ids = []string{it.info.ID}
				}
			}
			if len(ids) == 0 {
				return m, nil
			}
			m.confirmKillAll = false
			m.confirmKillIDs = append([]string(nil), ids...)
			m.confirmKillPrompt = killSessionsConfirmPrompt(m.sessions, ids)
			m.screen = screenConfirmKillSessions
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.sessions, cmd = m.sessions.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	w, h := clampDims(m.width, m.height)
	errMsg := ""
	if m.err != nil {
		errMsg = m.err.Error()
	}

	switch m.screen {
	case screenForm:
		bottom := formBottomBar(w, m.form.stepIndex+1, len(m.form.steps))
		body := lipgloss.Place(w, max(4, h-headerHeight(w)-bottomBarHeight(w, bottom)), lipgloss.Center, lipgloss.Center, m.form.View())
		return layoutScreen(w, h, body, bottom, errMsg)
	case screenConfirmDelete:
		bottom := confirmBottomBar(w, m.confirmAlias)
		body := confirmStyle.Width(max(40, w-8)).Render(fmt.Sprintf("Delete profile %q? (y/n)", m.confirmAlias))
		body = lipgloss.Place(w, max(4, h-headerHeight(w)-bottomBarHeight(w, bottom)), lipgloss.Center, lipgloss.Center, body)
		return layoutScreen(w, h, body, bottom, errMsg)
	case screenConfirmKillSessions:
		bottom := killConfirmBottomBar(w, m.confirmKillPrompt)
		body := confirmStyle.Width(max(40, w-8)).Render(m.confirmKillPrompt)
		body = lipgloss.Place(w, max(4, h-headerHeight(w)-bottomBarHeight(w, bottom)), lipgloss.Center, lipgloss.Center, body)
		return layoutScreen(w, h, body, bottom, errMsg)
	case screenSessions:
		return layoutScreen(w, h, m.sessions.View(), m.sessionsBottomBar(), errMsg)
	default:
		return layoutScreen(w, h, m.list.View(), m.listBottomBar(), errMsg)
	}
}

func (m Model) refreshLayout() tea.Cmd {
	w, h := m.width, m.height
	return func() tea.Msg {
		return tea.WindowSizeMsg{Width: w, Height: h}
	}
}

func (m Model) connectCmd(alias string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return connectDoneMsg{alias: alias, err: err}
		}
		err = connect.Profile(cfg, alias, connect.WithQuiet(true), connect.WithDetached(true))
		return connectDoneMsg{alias: alias, err: err}
	}
}

type connectDoneMsg struct {
	alias string
	err   error
}
type saveDoneMsg struct {
	cfg *config.Config
	err error
}

func findIndex(profiles []profile.Profile, alias string) int {
	for i, p := range profiles {
		if p.Alias == alias {
			return i
		}
	}
	return -1
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	footerStyle  = lipgloss.NewStyle().MarginTop(1)
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	confirmStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)
)

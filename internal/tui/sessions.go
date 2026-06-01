package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mewisme/mewsh/internal/connect"
)

type sessionItem struct {
	info   connect.SessionInfo
	marked bool
}

func (i sessionItem) FilterValue() string {
	return i.info.Alias + " " + i.info.Hostname + " " + i.info.Target + " " + i.info.ID
}

func (i sessionItem) Title() string {
	if i.marked {
		return "✓ " + i.info.Alias
	}
	return i.info.Alias
}

func (i sessionItem) Description() string {
	parts := []string{i.info.Target}
	if i.info.Hostname != "" {
		parts = append(parts, "tunnel: "+i.info.Hostname)
	}
	if i.info.PID > 0 {
		parts = append(parts, fmt.Sprintf("pid %d", i.info.PID))
	}
	return strings.Join(parts, " · ")
}

type sessionsListModel struct {
	list   list.Model
	items  []sessionItem
	marked map[string]bool
}

func newSessionsListModel() sessionsListModel {
	m := sessionsListModel{marked: map[string]bool{}}
	return m.refresh()
}

func (m sessionsListModel) refresh() sessionsListModel {
	sessions := connect.ListSessions()
	m.items = make([]sessionItem, len(sessions))
	for i, s := range sessions {
		m.items[i] = sessionItem{info: s, marked: m.marked[s.ID]}
	}
	selected := m.list.Index()
	m.list = m.buildList()
	if selected >= 0 && selected < len(m.items) {
		m.list.Select(selected)
	}
	return m
}

func (m sessionsListModel) buildList() list.Model {
	listItems := make([]list.Item, len(m.items))
	for i, it := range m.items {
		listItems[i] = it
	}

	w, h := m.list.Width(), m.list.Height()
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 20
	}

	delegate := newTwoLineDelegate()

	l := list.New(listItems, delegate, w, h)
	configureList(&l, "Active Sessions", "session", "sessions", w)
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "mark")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "kill")),
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "all")),
			key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "back")),
		}
	}
	return l
}

func (m sessionsListModel) withSize(w, h int) sessionsListModel {
	w = max(20, w)
	h = max(4, h)
	m.list.SetSize(w, h)
	m.list.Styles.TitleBar = m.list.Styles.TitleBar.Width(w)
	m.list.Styles.HelpStyle = m.list.Styles.HelpStyle.Width(w)
	return m
}

func (m sessionsListModel) filterInputActive() bool {
	return m.list.FilterState() == list.Filtering
}

func (m sessionsListModel) Update(msg tea.Msg) (sessionsListModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m sessionsListModel) View() string {
	if len(m.list.Items()) == 0 {
		return lipgloss.NewStyle().
			Width(m.list.Width()).
			Height(m.list.Height()).
			Padding(1, 2).
			Render("No active sessions.")
	}
	return m.list.View()
}

func (m *sessionsListModel) selectedItem() *sessionItem {
	item, ok := m.list.SelectedItem().(sessionItem)
	if !ok {
		return nil
	}
	return &item
}

func (m sessionsListModel) toggleMark() sessionsListModel {
	it := m.selectedItem()
	if it == nil {
		return m
	}
	if m.marked[it.info.ID] {
		delete(m.marked, it.info.ID)
	} else {
		m.marked[it.info.ID] = true
	}
	return m.refresh()
}

func (m sessionsListModel) markedIDs() []string {
	if len(m.marked) == 0 {
		return nil
	}
	ids := make([]string, 0, len(m.marked))
	for id := range m.marked {
		ids = append(ids, id)
	}
	return ids
}

func (m sessionsListModel) markedCount() int {
	return len(m.marked)
}

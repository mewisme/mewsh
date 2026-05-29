package tui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mewisme/mewsh/internal/profile"
)

type profileItem struct {
	profile profile.Profile
}

func (i profileItem) FilterValue() string {
	return i.profile.Alias + " " + i.profile.Host + " " + i.profile.CFHostname + " " + i.profile.User + " " + i.profile.Note
}

func (i profileItem) Title() string { return i.profile.Alias }

func (i profileItem) Description() string { return i.profile.Summary() }

type listModel struct {
	list     list.Model
	profiles []profile.Profile
}

func newListModel(profiles []profile.Profile) listModel {
	m := listModel{profiles: profiles}
	m.list = m.buildList()
	return m
}

func (m listModel) buildList() list.Model {
	items := make([]list.Item, len(m.profiles))
	for i, p := range m.profiles {
		items[i] = profileItem{profile: p}
	}

	w, h := m.list.Width(), m.list.Height()
	if w < 1 {
		w = 80
	}
	if h < 1 {
		h = 20
	}

	delegate := newTwoLineDelegate()

	l := list.New(items, delegate, w, h)
	l.Title = "Profiles"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	// Center title via TitleBar only — do not set Width on Title or bubbles
	// appends "  "+status and truncates, which shows "..." at end of line.
	l.Styles.TitleBar = lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Padding(0, 0, 1, 0)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	l.Styles.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	l.SetShowHelp(false)
	return l
}

func (m listModel) withSize(w, h int) listModel {
	w = max(20, w)
	h = max(4, h)
	m.list.SetSize(w, h)
	m.list.Styles.TitleBar = m.list.Styles.TitleBar.Width(w)
	return m
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	if len(m.list.Items()) == 0 {
		return lipgloss.NewStyle().
			Width(m.list.Width()).
			Height(m.list.Height()).
			Padding(1, 2).
			Render("No profiles yet. Press 'm' then 'a' to add one.")
	}
	return m.list.View()
}

func (m listModel) filterInputActive() bool {
	return m.list.FilterState() == list.Filtering
}

func (m listModel) selectedProfile() *profile.Profile {
	item, ok := m.list.SelectedItem().(profileItem)
	if !ok {
		return nil
	}
	p := item.profile
	return &p
}

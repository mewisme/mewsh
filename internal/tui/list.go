package tui

import (
	"github.com/charmbracelet/bubbles/key"
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
	configureList(&l, "Profiles", "profile", "profiles", w)
	return l
}

// configureList enables the full bubbles/list feature set for mewsh screens.
// Quit is handled by the root model; use NewStatusMessage / StartSpinner from the parent when needed.
func configureList(l *list.Model, title, itemSingular, itemPlural string, w int) {
	l.Title = title
	l.InfiniteScrolling = true
	l.SetFilteringEnabled(true)
	l.SetShowTitle(true)
	l.SetShowFilter(true)
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetShowHelp(true)
	l.SetStatusBarItemName(itemSingular, itemPlural)
	l.DisableQuitKeybindings()
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	l.Help.Ellipsis = ""

	subdued := lipgloss.Color("241")
	// Center title via TitleBar only — do not set Width on Title or bubbles
	// appends "  "+status and truncates, which shows "..." at end of line.
	l.Styles.TitleBar = lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Padding(0, 0, 1, 0)
	l.Styles.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	l.Styles.Spinner = lipgloss.NewStyle().Foreground(subdued)
	l.Styles.StatusBar = lipgloss.NewStyle().Foreground(subdued).Padding(0, 0, 0, 0)
	l.Styles.StatusEmpty = lipgloss.NewStyle().Foreground(subdued)
	l.Styles.StatusBarFilterCount = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	l.Styles.NoItems = lipgloss.NewStyle().Foreground(subdued)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(subdued)
	l.Styles.HelpStyle = lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Foreground(subdued)
	l.Styles.ActivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	l.Styles.InactivePaginationDot = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
}

func (m listModel) withMenuOpen(open bool) listModel {
	m.list.AdditionalShortHelpKeys = func() []key.Binding {
		if open {
			return []key.Binding{
				key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
				key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
				key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
				key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sessions")),
				key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "back")),
			}
		}
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect")),
			key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "menu")),
		}
	}
	return m
}

func (m listModel) withSize(w, h int) listModel {
	w = max(20, w)
	h = max(4, h)
	m.list.SetSize(w, h)
	m.list.Styles.TitleBar = m.list.Styles.TitleBar.Width(w)
	m.list.Styles.HelpStyle = m.list.Styles.HelpStyle.Width(w)
	return m
}

func (m listModel) Update(msg tea.Msg) (listModel, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) statusMessage(s string) tea.Cmd {
	return m.list.NewStatusMessage(s)
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

package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	profileItemPadLeft = 2
	titleDescGap       = 0
)

// twoLineDelegate renders alias and description in one block (shared left edge)
// with a gap between the two lines; selected rows use a single left border.
type twoLineDelegate struct {
	list.DefaultDelegate
}

func newTwoLineDelegate() twoLineDelegate {
	d := twoLineDelegate{DefaultDelegate: list.NewDefaultDelegate()}
	d.ShowDescription = true
	d.SetHeight(2 + titleDescGap)
	return d
}

func (d twoLineDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	di, ok := item.(list.DefaultItem)
	if !ok {
		d.DefaultDelegate.Render(w, m, index, item)
		return
	}

	title := di.Title()
	desc := di.Description()
	if m.Width() <= 0 {
		return
	}

	textwidth := m.Width() - profileItemPadLeft
	title = ansi.Truncate(title, textwidth, "…")
	if d.ShowDescription {
		var lines []string
		for i, line := range strings.Split(desc, "\n") {
			if i >= 1 {
				break
			}
			lines = append(lines, ansi.Truncate(line, textwidth, "…"))
		}
		desc = strings.Join(lines, "\n")
	}

	isSelected := index == m.Index() && m.FilterState() != list.Filtering
	emptyFilter := m.FilterState() == list.Filtering && m.FilterValue() == ""
	isFiltered := m.FilterState() == list.Filtering || m.FilterState() == list.FilterApplied

	if isFiltered {
		matchedRunes := m.MatchesForItem(index)
		if len(matchedRunes) > 0 {
			unmatched := lipgloss.NewStyle().Inline(true)
			if isSelected {
				unmatched = unmatched.Foreground(lipgloss.Color("170")).Bold(true)
			} else {
				unmatched = unmatched.Foreground(lipgloss.Color("252"))
			}
			matched := unmatched.Inherit(list.NewDefaultItemStyles().FilterMatch)
			title = lipgloss.StyleRunes(title, matchedRunes, matched, unmatched)
		}
	}

	var titleStyle, descStyle lipgloss.Style
	switch {
	case emptyFilter:
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).MarginTop(titleDescGap)
	case isSelected:
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("170")).Bold(true)
		descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).MarginTop(titleDescGap)
	default:
		titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
		descStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(titleDescGap)
	}

	var content string
	if d.ShowDescription {
		content = lipgloss.JoinVertical(lipgloss.Left,
			titleStyle.Render(title),
			descStyle.Render(desc),
		)
	} else {
		content = titleStyle.Render(title)
	}

	var out string
	if isSelected {
		out = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("170")).
			PaddingLeft(1).
			Render(content)
	} else {
		out = lipgloss.NewStyle().PaddingLeft(profileItemPadLeft).Render(content)
	}

	fmt.Fprint(w, out)
}

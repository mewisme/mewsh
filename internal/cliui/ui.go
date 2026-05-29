// Package cliui provides consistent colors, prefixes, and tables for mewsh CLI output.
package cliui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Level selects the line prefix and color.
type Level int

const (
	LevelInfo Level = iota
	LevelOK
	LevelWarn
	LevelErr
	LevelDim
	LevelLabel
	LevelCmd
)

// Enabled reports whether styling is active (respects NO_COLOR).
func Enabled() bool {
	return os.Getenv("NO_COLOR") == ""
}

var (
	styleInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	styleOK    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	styleErr   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleDim   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
	styleCmd   = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	styleHead  = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Bold(true)
)

type prefix struct {
	glyph string
	plain string
	style lipgloss.Style
}

func prefixFor(level Level) prefix {
	switch level {
	case LevelOK:
		return prefix{"✓", "ok", styleOK}
	case LevelWarn:
		return prefix{"!", "warn", styleWarn}
	case LevelErr:
		return prefix{"✗", "err", styleErr}
	case LevelDim:
		return prefix{"·", "·", styleDim}
	case LevelLabel:
		return prefix{"", "", styleLabel}
	case LevelCmd:
		return prefix{"$", ">", styleCmd}
	default:
		return prefix{"●", "info", styleInfo}
	}
}

// Line writes one prefixed line to w.
func Line(w io.Writer, level Level, msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	p := prefixFor(level)
	if !Enabled() {
		if p.glyph != "" {
			fmt.Fprintf(w, "%s %s\n", p.plain, msg)
		} else {
			fmt.Fprintf(w, "%s\n", msg)
		}
		return
	}
	if p.glyph != "" {
		fmt.Fprintf(w, "%s %s\n", p.style.Render(p.glyph), paint(level, msg))
	} else {
		fmt.Fprintf(w, "%s\n", paint(level, msg))
	}
}

func paint(level Level, msg string) string {
	switch level {
	case LevelOK:
		return styleOK.Render(msg)
	case LevelWarn:
		return styleWarn.Render(msg)
	case LevelErr:
		return styleErr.Render(msg)
	case LevelDim:
		return styleDim.Render(msg)
	case LevelLabel:
		return styleLabel.Render(msg)
	case LevelCmd:
		return styleCmd.Render(msg)
	default:
		return msg
	}
}

// Lines writes multiple prefixed lines (one per non-empty line in msg).
func Lines(w io.Writer, level Level, msg string) {
	for _, line := range strings.Split(msg, "\n") {
		Line(w, level, line)
	}
}

// Labeled writes "key: value" with styled key.
func Labeled(w io.Writer, key, value string) {
	if !Enabled() {
		fmt.Fprintf(w, "%s: %s\n", key, value)
		return
	}
	fmt.Fprintf(w, "%s %s\n", styleLabel.Render(key+":"), paint(LevelDim, value))
}

// Block writes an indented block (commands, notes).
func Block(w io.Writer, level Level, lines ...string) {
	indent := "  "
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		p := prefixFor(level)
		if !Enabled() {
			fmt.Fprintf(w, "%s%s\n", indent, line)
			continue
		}
		if level == LevelCmd {
			fmt.Fprintf(w, "%s%s\n", indent, styleCmd.Render(line))
		} else {
			fmt.Fprintf(w, "%s%s %s\n", indent, p.style.Render(p.glyph), paint(level, line))
		}
	}
}

// PrintKV writes aligned key: value lines (no table header).
func PrintKV(w io.Writer, pairs [][2]string) {
	if len(pairs) == 0 {
		return
	}
	maxKey := 0
	for _, p := range pairs {
		if len(p[0]) > maxKey {
			maxKey = len(p[0])
		}
	}
	for _, p := range pairs {
		key := p[0]
		val := p[1]
		if val == "" {
			continue
		}
		if !Enabled() {
			fmt.Fprintf(w, "%-*s  %s\n", maxKey, key+":", val)
			continue
		}
		fmt.Fprintf(w, "%s %s\n",
			styleLabel.Render(fmt.Sprintf("%-*s:", maxKey, key)),
			paint(LevelDim, val),
		)
	}
}

// Section writes a small titled section header.
func Section(w io.Writer, title string) {
	if !Enabled() {
		fmt.Fprintf(w, "%s\n", title)
		return
	}
	fmt.Fprintf(w, "%s\n", styleHead.Render(title))
}

// PrintTable renders a column-aligned table to w.
func PrintTable(w io.Writer, header []string, rows [][]string) {
	if len(header) == 0 {
		return
	}
	widths := make([]int, len(header))
	for i, h := range header {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	format := tableFormat(widths)
	if Enabled() {
		fmt.Fprintln(w, styleHead.Render(fmt.Sprintf(format, stringsToAny(header)...)))
	} else {
		fmt.Fprintf(w, format+"\n", stringsToAny(header)...)
	}
	for _, row := range rows {
		vals := make([]string, len(header))
		for i := range header {
			if i < len(row) {
				vals[i] = row[i]
			}
		}
		fmt.Fprintf(w, format+"\n", stringsToAny(vals)...)
	}
}

func tableFormat(widths []int) string {
	parts := make([]string, len(widths))
	for i, w := range widths {
		parts[i] = fmt.Sprintf("%%-%ds", w)
	}
	return strings.Join(parts, " ")
}

func stringsToAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// Stdout helpers.
func Info(msg string) { Line(os.Stdout, LevelInfo, msg) }
func OK(msg string)   { Line(os.Stdout, LevelOK, msg) }
func Warn(msg string) { Line(os.Stdout, LevelWarn, msg) }
func Err(msg string)  { Line(os.Stderr, LevelErr, msg) }
func Dim(msg string)  { Line(os.Stdout, LevelDim, msg) }

func Infof(w io.Writer, format string, args ...any) { Line(w, LevelInfo, fmt.Sprintf(format, args...)) }
func OKf(w io.Writer, format string, args ...any)   { Line(w, LevelOK, fmt.Sprintf(format, args...)) }
func Warnf(w io.Writer, format string, args ...any) { Line(w, LevelWarn, fmt.Sprintf(format, args...)) }
func Errf(w io.Writer, format string, args ...any)  { Line(w, LevelErr, fmt.Sprintf(format, args...)) }

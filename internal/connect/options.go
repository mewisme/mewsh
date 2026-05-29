package connect

import (
	"io"
	"strings"

	"github.com/mewisme/mewsh/internal/cliui"
)

type Options struct {
	status     io.Writer
	quiet      bool
	detached   bool
	background bool
}

type Option func(*Options)

func defaultOptions() Options {
	return Options{}
}

func WithQuiet(quiet bool) Option {
	return func(o *Options) {
		o.quiet = quiet
	}
}

func WithStatus(w io.Writer) Option {
	return func(o *Options) {
		o.status = w
	}
}

// WithDetached spawns SSH in a separate terminal and returns immediately
// (used by the TUI so the profile list stays interactive).
func WithDetached(detached bool) Option {
	return func(o *Options) {
		o.detached = detached
	}
}

// WithBackground starts a detached worker that runs SSH without a GUI terminal
// and survives when the invoking shell exits (headless servers, scripts).
func WithBackground(background bool) Option {
	return func(o *Options) {
		o.background = background
	}
}

func (o Options) say(msg string) {
	if o.quiet || o.status == nil {
		return
	}
	msg = strings.TrimRight(msg, "\n")
	if msg == "" {
		return
	}
	for _, line := range strings.Split(msg, "\n") {
		cliui.Line(o.status, cliui.LevelInfo, line)
	}
}

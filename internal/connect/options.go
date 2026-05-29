package connect

import "io"

type Options struct {
	status   io.Writer
	quiet    bool
	detached bool
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

func (o Options) say(msg string) {
	if o.quiet || o.status == nil {
		return
	}
	_, _ = io.WriteString(o.status, msg)
}

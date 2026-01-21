package log

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// GithubHandler is a slog.Handler that outputs log messages using GitHub Actions
// workflow command format. Debug, warning, and error levels use the ::command::
// syntax, while info level outputs plain text.
type GithubHandler struct {
	out    io.Writer
	mu     *sync.Mutex
	groups []string
	attrs  []slog.Attr
}

// NewGithubHandler creates a new GithubHandler that writes to w.
func NewGithubHandler(w io.Writer) *GithubHandler {
	return &GithubHandler{
		out: w,
		mu:  &sync.Mutex{},
	}
}

// Enabled reports whether the handler handles records at the given level.
// All levels are enabled for GitHub Actions logging.
func (h *GithubHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

// Handle formats the record using GitHub Actions workflow commands and writes it.
func (h *GithubHandler) Handle(_ context.Context, r slog.Record) error {
	buf := make([]byte, 0, 256)

	// Format based on level
	switch {
	case r.Level < slog.LevelInfo:
		buf = append(buf, "::debug::"...)
	case r.Level < slog.LevelWarn:
		// Info level: plain text, no prefix
	case r.Level < slog.LevelError:
		buf = append(buf, "::warning::"...)
	default:
		buf = append(buf, "::error::"...)
	}

	// Write the message
	buf = append(buf, r.Message...)

	// Write pre-collected attrs from WithAttrs
	for _, a := range h.attrs {
		buf = h.appendAttr(buf, a)
	}

	// Write record attrs
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})

	buf = append(buf, '\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

// WithAttrs returns a new handler with the given attributes added.
func (h *GithubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &GithubHandler{
		out:    h.out,
		mu:     h.mu,
		groups: h.groups,
		attrs:  newAttrs,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *GithubHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)
	return &GithubHandler{
		out:    h.out,
		mu:     h.mu,
		groups: newGroups,
		attrs:  h.attrs,
	}
}

// appendAttr appends a single attribute to the buffer in key=value format.
func (h *GithubHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}

	buf = append(buf, ' ')

	// Prepend group names if any
	for _, g := range h.groups {
		buf = append(buf, g...)
		buf = append(buf, '.')
	}

	buf = append(buf, a.Key...)
	buf = append(buf, '=')
	buf = appendValue(buf, a.Value)
	return buf
}

package log

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// PlainHandler is a slog.Handler that outputs log messages in plain text format.
// It outputs only the message and attributes as key=value pairs, without
// timestamp or level information.
type PlainHandler struct {
	out    io.Writer
	mu     *sync.Mutex
	level  slog.Leveler
	groups []string
	attrs  []slog.Attr
}

// PlainHandlerOptions are options for a PlainHandler.
type PlainHandlerOptions struct {
	// Level is the minimum level to log. If nil, defaults to slog.LevelInfo.
	Level slog.Leveler
}

// NewPlainHandler creates a new PlainHandler that writes to w.
func NewPlainHandler(w io.Writer, opts *PlainHandlerOptions) *PlainHandler {
	h := &PlainHandler{
		out: w,
		mu:  &sync.Mutex{},
	}
	if opts != nil && opts.Level != nil {
		h.level = opts.Level
	}
	return h
}

// Enabled reports whether the handler handles records at the given level.
func (h *PlainHandler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.level != nil {
		minLevel = h.level.Level()
	}
	return level >= minLevel
}

// Handle formats the record as plain text and writes it to the output.
func (h *PlainHandler) Handle(_ context.Context, r slog.Record) error {
	buf := make([]byte, 0, 256)

	// Write level prefix for non-info levels
	if r.Level != slog.LevelInfo {
		buf = append(buf, '[')
		buf = append(buf, r.Level.String()...)
		buf = append(buf, "] "...)
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
func (h *PlainHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &PlainHandler{
		out:    h.out,
		mu:     h.mu,
		level:  h.level,
		groups: h.groups,
		attrs:  newAttrs,
	}
}

// WithGroup returns a new handler with the given group name.
func (h *PlainHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newGroups := make([]string, len(h.groups), len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups = append(newGroups, name)
	return &PlainHandler{
		out:    h.out,
		mu:     h.mu,
		level:  h.level,
		groups: newGroups,
		attrs:  h.attrs,
	}
}

// appendAttr appends a single attribute to the buffer in key=value format.
func (h *PlainHandler) appendAttr(buf []byte, a slog.Attr) []byte {
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

// appendValue appends the value to the buffer.
func appendValue(buf []byte, v slog.Value) []byte {
	switch v.Kind() {
	case slog.KindString:
		buf = append(buf, v.String()...)
	case slog.KindGroup:
		attrs := v.Group()
		for i, a := range attrs {
			if i > 0 {
				buf = append(buf, ' ')
			}
			buf = append(buf, a.Key...)
			buf = append(buf, '=')
			buf = appendValue(buf, a.Value.Resolve())
		}
	default:
		buf = append(buf, v.String()...)
	}
	return buf
}

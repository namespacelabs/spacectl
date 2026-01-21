package log_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/namespacelabs/space/internal/log"
)

func TestPlainHandler_MessageOnly(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewPlainHandler(&buf, nil))

	logger.Info("hello world")

	got := buf.String()
	want := "hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlainHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewPlainHandler(&buf, nil))

	logger.Info("mounting path", slog.String("from", "/cache/dir"), slog.String("to", "/target"))

	got := buf.String()
	want := "mounting path from=/cache/dir to=/target\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlainHandler_WithIntAttr(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewPlainHandler(&buf, nil))

	logger.Info("cache hit rate", slog.Int("hits", 5), slog.Int("total", 10))

	got := buf.String()
	want := "cache hit rate hits=5 total=10\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlainHandler_LoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewPlainHandler(&buf, nil))
	logger = logger.With(slog.String("component", "cache"))

	logger.Info("mounted")

	got := buf.String()
	want := "mounted component=cache\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlainHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewPlainHandler(&buf, nil))
	logger = logger.WithGroup("mount")

	logger.Info("path mounted", slog.String("target", "/cache"))

	got := buf.String()
	want := "path mounted mount.target=/cache\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlainHandler_DefaultLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := log.NewPlainHandler(&buf, nil)

	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected debug to be disabled by default")
	}
	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected info to be enabled by default")
	}
}

func TestPlainHandler_CustomLevel(t *testing.T) {
	var buf bytes.Buffer
	handler := log.NewPlainHandler(&buf, &log.PlainHandlerOptions{
		Level: slog.LevelDebug,
	})

	if !handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("expected debug to be enabled")
	}
}

func TestPlainHandler_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewPlainHandler(&buf, &log.PlainHandlerOptions{
		Level: slog.LevelWarn,
	}))

	logger.Info("should not appear")
	logger.Warn("should appear")

	got := buf.String()
	want := "[WARN] should appear\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPlainHandler_LevelPrefix(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{slog.LevelDebug, "[DEBUG] test\n"},
		{slog.LevelInfo, "test\n"},
		{slog.LevelWarn, "[WARN] test\n"},
		{slog.LevelError, "[ERROR] test\n"},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(log.NewPlainHandler(&buf, &log.PlainHandlerOptions{
				Level: slog.LevelDebug,
			}))

			logger.Log(context.Background(), tt.level, "test")

			got := buf.String()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

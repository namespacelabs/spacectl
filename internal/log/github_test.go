package log_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/namespacelabs/space/internal/log"
)

func TestGithubHandler_InfoPlainText(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))

	logger.Info("hello world")

	got := buf.String()
	want := "hello world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGithubHandler_DebugFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))

	logger.Debug("debug message")

	got := buf.String()
	want := "::debug::debug message\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGithubHandler_WarnFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))

	logger.Warn("warning message")

	got := buf.String()
	want := "::warning::warning message\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGithubHandler_ErrorFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))

	logger.Error("error message")

	got := buf.String()
	want := "::error::error message\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGithubHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))

	logger.Info("mounting path", slog.String("from", "/cache"), slog.String("to", "/target"))

	got := buf.String()
	want := "mounting path from=/cache to=/target\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGithubHandler_LoggerWith(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))
	logger = logger.With(slog.String("component", "cache"))

	logger.Info("mounted")

	got := buf.String()
	want := "mounted component=cache\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGithubHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))
	logger = logger.WithGroup("mount")

	logger.Info("path mounted", slog.String("target", "/cache"))

	got := buf.String()
	want := "path mounted mount.target=/cache\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGithubHandler_AllLevelsEnabled(t *testing.T) {
	var buf bytes.Buffer
	handler := log.NewGithubHandler(&buf)

	levels := []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	for _, lvl := range levels {
		if !handler.Enabled(context.Background(), lvl) {
			t.Errorf("expected %s to be enabled", lvl)
		}
	}
}

func TestGithubHandler_ErrorWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(log.NewGithubHandler(&buf))

	logger.Error("failed to mount", slog.String("path", "/cache"), slog.Int("code", 1))

	got := buf.String()
	want := "::error::failed to mount path=/cache code=1\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

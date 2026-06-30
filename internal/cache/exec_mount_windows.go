//go:build windows

package cache

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func mount(ctx context.Context, from, to string) error {
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return fmt.Errorf("creating parent of to path %q: %w", to, err)
	}

	if err := os.RemoveAll(to); err != nil {
		return fmt.Errorf("removing existing to path %q: %w", to, err)
	}

	if _, err := run(ctx, "cmd", "/c", "mklink", "/J", to, from); err != nil {
		return fmt.Errorf("creating junction from %q to %q: %w", to, from, err)
	}

	return nil
}

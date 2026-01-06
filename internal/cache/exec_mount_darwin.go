//go:build darwin

package cache

import (
	"context"
	"fmt"
	"path/filepath"
)

func mount(ctx context.Context, from, to string) error {
	if err := sudoMkdirP(ctx, filepath.Dir(to)); err != nil {
		return err
	}

	if _, err := run(ctx, "sudo", "rm", "-rf", to); err != nil {
		return fmt.Errorf("removing to path %q: %w", to, err)
	}

	if _, err := run(ctx, "sudo", "ln", "-sfn", from, to); err != nil {
		return fmt.Errorf("symlinking from %q to %q: %w", from, to, err)
	}

	return chownSelf(ctx, to)
}

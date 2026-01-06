//go:build !darwin

package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
)

func mount(ctx context.Context, from, to string) error {
	// existing files can't be mounted over, so we'll need to remove first
	mountPathInfo, err := os.Lstat(to)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stating to path %q: %w", to, err)
	}
	if mountPathInfo != nil && !mountPathInfo.IsDir() {
		if _, err := run(ctx, "sudo", "rm", "-rf", to); err != nil {
			return fmt.Errorf("removing non-directory to path %q: %w", to, err)
		}
	}

	if err := sudoMkdirP(ctx, to); err != nil {
		return err
	}

	if _, err := run(ctx, "sudo", "mount", "--bind", from, to); err != nil {
		return fmt.Errorf("binding from %q to %q: %w", from, to, err)
	}

	return nil
}

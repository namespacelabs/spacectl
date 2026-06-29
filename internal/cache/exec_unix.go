//go:build linux || darwin

package cache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func (e DefaultExecutor) RemoveAll(name string) error {
	_, err := run(context.Background(), "sudo", "rm", "-rf", name)
	return err
}

func (e DefaultExecutor) DiskUsage(ctx context.Context, path string) (DiskUsage, error) {
	output, err := run(ctx, "df", "-h", path)
	if err != nil {
		return DiskUsage{}, fmt.Errorf("running df: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return DiskUsage{}, errors.New("unexpected df output: missing data line")
	}

	columns := strings.Fields(lines[1])
	if len(columns) < 3 {
		return DiskUsage{}, errors.New("unexpected df output: insufficient columns")
	}

	return DiskUsage{
		Total: columns[1],
		Used:  columns[2],
	}, nil
}

// chownSelf changes the ownership of the given path to the current user.
func chownSelf(ctx context.Context, path string) error {
	currentUser, err := user.Current()
	if err != nil {
		return fmt.Errorf("getting current user: %w", err)
	}

	_, err = run(ctx, "sudo", "chown", fmt.Sprintf("%s:%s", currentUser.Uid, currentUser.Gid), path)
	if err != nil {
		return fmt.Errorf("sudo chown failed: %w", err)
	}

	return nil
}

// sudoMkdirP creates all ancestor directories of the given path using sudo.
func sudoMkdirP(ctx context.Context, path string) error {
	for _, p := range ancestors(path) {
		// Check if directory already exists
		_, err := os.Stat(p)
		if err == nil {
			// Directory exists, continue to next
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			// Some other error occurred
			return fmt.Errorf("stat %q: %w", p, err)
		}

		// Directory doesn't exist, try to create it
		if _, err := run(ctx, "sudo", "mkdir", p); err != nil {
			return fmt.Errorf("sudo mkdir directory `%s`: %w", p, err)
		}

		// Change ownership to current user
		if err := chownSelf(ctx, p); err != nil {
			return fmt.Errorf("chown %q: %w", p, err)
		}
	}

	return nil
}

// ancestors returns all ancestor directories of the given path, from root to the path itself.
func ancestors(path string) []string {
	var result []string
	for {
		parent := filepath.Dir(path)
		if parent == path {
			break
		}
		result = append(result, path)
		path = parent
	}

	// Reverse to get root-to-leaf order
	for i := len(result)/2 - 1; i >= 0; i-- {
		opp := len(result) - 1 - i
		result[i], result[opp] = result[opp], result[i]
	}

	return result
}

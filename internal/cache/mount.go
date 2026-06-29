//go:generate moq -out mount_mock.go . Executor
package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/namespacelabs/spacectl/internal/cache/mode"
)

type MountRequest struct {
	DetectAllModes bool
	DetectModes    []string
	ManualModes    []string
	ManualPaths    []string
}

// EnabledModes returns the set of enabled cache modes based on the request.
// It performs detection as necessary, based on the detect modes specified.
func (req MountRequest) EnabledModes(ctx context.Context, available mode.Modes) (mode.Modes, error) {
	if !req.DetectAllModes && len(req.DetectModes) == 0 && len(req.ManualModes) == 0 && len(req.ManualPaths) == 0 {
		return nil, errors.New("at least one cache mode or path must be specified")
	}

	enabled := req.ManualModes
	detect := req.DetectModes
	if req.DetectAllModes {
		detect = available.Names()
	}
	if len(detect) > 0 {
		filtered, err := available.Filter(detect)
		if err != nil {
			return nil, err
		}

		detected, err := filtered.Detect(ctx, mode.DetectRequest{
			Exec: mode.DefaultExecutor{},
		})
		if err != nil {
			return nil, err
		}

		enabled = append(enabled, detected.Names()...)
	}

	return available.Filter(enabled)
}

type MountResponse struct {
	Input  MountResponseInput  `json:"input,omitzero"`
	Output MountResponseOutput `json:"output,omitzero"`
}

type MountResponseInput struct {
	Modes []string `json:"modes,omitzero"`
	Paths []string `json:"paths,omitzero"`
}

type MountResponseOutput struct {
	DestructiveMode bool              `json:"destructive_mode"`
	AddEnvs         map[string]string `json:"add_envs,omitzero"`
	DiskUsage       *DiskUsage        `json:"disk_usage,omitzero"` // lookup can fail, so inclusion is optional
	Mounts          []MountResult     `json:"mounts,omitzero"`
	RemovedPaths    []string          `json:"removed_paths,omitzero"`
}

type MountResult struct {
	Mode      string `json:"mode,omitzero"`
	CachePath string `json:"cache_path"`
	MountPath string `json:"mount_path"`
	CacheHit  bool   `json:"cache_hit"`
}

type CacheMetadata struct {
	UpdatedAt   string                        `json:"updatedAt"`
	Version     int                           `json:"version"`
	UserRequest map[string]CacheMetadataEntry `json:"userRequest"`
}

type CacheMetadataEntry struct {
	CacheFramework *string  `json:"cacheFramework"`
	MountTarget    []string `json:"mountTarget"`
	Source         string   `json:"source"`
}

func NewMounter(cacheRoot string) (Mounter, error) {
	cacheRoot, err := absDir(cacheRoot)
	if err != nil {
		return Mounter{}, fmt.Errorf("resolving cache root: %w", err)
	}

	return Mounter{
		CacheRoot: cacheRoot,
		Exec:      DefaultExecutor{},
		Modes:     mode.DefaultModes(),
	}, nil
}

type Mounter struct {
	DestructiveMode bool
	CacheRoot       string
	Exec            Executor
	Modes           mode.Modes
}

// Mount mounts the cache paths based on the given request.
func (m Mounter) Mount(ctx context.Context, req MountRequest) (MountResponse, error) {
	result := MountResponse{
		Output: MountResponseOutput{
			DestructiveMode: m.DestructiveMode,
		},
	}

	// Mount modes
	modes, err := req.EnabledModes(ctx, m.Modes)
	if err != nil {
		return MountResponse{}, err
	}
	if err := m.mountModes(ctx, modes, &result); err != nil {
		return MountResponse{}, err
	}

	// Mount manual paths
	if err := m.mountPaths(ctx, req.ManualPaths, &result); err != nil {
		return MountResponse{}, err
	}

	// Get disk usage (allowed to fail)
	if usage, err := m.Exec.DiskUsage(ctx, m.CacheRoot); err == nil {
		result.Output.DiskUsage = &usage
	}

	return result, nil
}

func (m Mounter) mountModes(ctx context.Context, modes mode.Modes, result *MountResponse) error {
	result.Input.Modes = modes.Names()

	plan, err := modes.Plan(ctx, mode.PlanRequest{CacheRoot: m.CacheRoot})
	if err != nil {
		return err
	}

	for modeName, p := range plan {
		for k, v := range p.AddEnvs {
			if result.Output.AddEnvs == nil {
				result.Output.AddEnvs = make(map[string]string)
			}
			result.Output.AddEnvs[k] = v
		}

		for _, subdir := range p.CacheDirs {
			mount, err := m.cacheDir(modeName, subdir)
			if err != nil {
				return fmt.Errorf("creating cache dir %q: %w", subdir, err)
			}
			result.Output.Mounts = append(result.Output.Mounts, mount)
		}

		for _, path := range p.MountPaths {
			mount, err := m.mountPath(ctx, modeName, path)
			if err != nil {
				return fmt.Errorf("mounting mode path %q: %w", path, err)
			}
			result.Output.Mounts = append(result.Output.Mounts, mount)
		}

		for _, path := range p.RemovePaths {
			if err := m.removePath(path, result); err != nil {
				return fmt.Errorf("removing mode path %q: %w", path, err)
			}
		}
	}

	return nil
}

func (m Mounter) mountPaths(ctx context.Context, paths []string, result *MountResponse) error {
	result.Input.Paths = append(result.Input.Paths, paths...)

	for _, path := range paths {
		mount, err := m.mountPath(ctx, "", path)
		if err != nil {
			return fmt.Errorf("mounting path %q: %w", path, err)
		}
		result.Output.Mounts = append(result.Output.Mounts, mount)
	}
	return nil
}

func (m Mounter) mountPath(ctx context.Context, modeName, path string) (MountResult, error) {
	path, err := resolveHome(path)
	if err != nil {
		return MountResult{}, fmt.Errorf("resolving path: %w", err)
	}

	cachePath := filepath.Join(m.CacheRoot, path)

	mount := MountResult{
		Mode:      modeName,
		CachePath: cachePath,
		MountPath: path,
	}

	_, err = m.Exec.Stat(cachePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return MountResult{}, fmt.Errorf("stat cache path %q: %w", cachePath, err)
	}
	mount.CacheHit = err == nil

	logAttrs := []any{slog.String("from", cachePath), slog.String("to", path)}
	if !m.DestructiveMode {
		slog.Debug("dry-run: would mount cache path", logAttrs...)
		return mount, nil
	}

	slog.Debug("mounting cache path", logAttrs...)

	if err := m.Exec.Mount(ctx, cachePath, path); err != nil {
		return MountResult{}, fmt.Errorf("mounting %q to %q: %w", cachePath, path, err)
	}
	return mount, nil
}

func (m Mounter) cacheDir(modeName, subdir string) (MountResult, error) {
	cachePath := filepath.Join(m.CacheRoot, subdir)

	mount := MountResult{
		Mode:      modeName,
		CachePath: cachePath,
		MountPath: cachePath,
	}

	_, err := m.Exec.Stat(cachePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return MountResult{}, fmt.Errorf("stat cache dir %q: %w", cachePath, err)
	}
	mount.CacheHit = err == nil

	if !m.DestructiveMode {
		slog.Debug("dry-run: would create cache dir", slog.String("path", cachePath))
		return mount, nil
	}

	slog.Debug("creating cache dir", slog.String("path", cachePath))

	if err := m.Exec.MkdirAll(cachePath, 0o755); err != nil {
		return MountResult{}, fmt.Errorf("creating cache dir %q: %w", cachePath, err)
	}
	return mount, nil
}

func (m Mounter) removePath(path string, result *MountResponse) error {
	result.Output.RemovedPaths = append(result.Output.RemovedPaths, path)

	if !m.DestructiveMode {
		slog.Debug("dry-run: would remove path", slog.String("path", path))
		return nil
	}

	slog.Debug("removing path", slog.String("path", path))

	if err := m.Exec.RemoveAll(path); err != nil {
		return fmt.Errorf("removing %q: %w", path, err)
	}
	return nil
}

type Executor interface {
	DiskUsage(ctx context.Context, path string) (DiskUsage, error)
	MkdirAll(path string, perm os.FileMode) error
	Mount(ctx context.Context, from, to string) error
	RemoveAll(name string) error
	Stat(name string) (os.FileInfo, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
}

type DiskUsage struct {
	Total string `json:"total"`
	Used  string `json:"used"`
}

type DefaultExecutor struct{}

func (e DefaultExecutor) Mount(ctx context.Context, from, to string) error {
	exists, err := MountTargetExists(to)
	if err != nil {
		return fmt.Errorf("checking mount target: %w", err)
	}
	if exists {
		slog.Debug("mount target will be overwritten", slog.String("path", to))
	}

	slog.Debug("mounting path", slog.String("from", from), slog.String("to", to))

	// create cache path, this is noop if it already exists
	if err := os.MkdirAll(from, 0o755); err != nil {
		return fmt.Errorf("creating from path %q: %w", from, err)
	}

	// os specific mount logic
	return mount(ctx, from, to)
}

func (e DefaultExecutor) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (e DefaultExecutor) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (e DefaultExecutor) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func absDir(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", path, err)
	}

	f, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("path %q does not exist", absPath)
		}
		return "", fmt.Errorf("stating path %q: %w", absPath, err)
	}
	if !f.IsDir() {
		return "", fmt.Errorf("path %q is not a directory", absPath)
	}

	return absPath, nil
}

// MountTargetExists checks if a mount target path exists and has content.
// For files, it returns true if the file exists.
// For directories, it returns true only if the directory is non-empty.
// For symlinks, it follows the link and applies the same logic.
// Returns false for non-existent paths or broken symlinks.
func MountTargetExists(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return symlinkTargetExists(path)
	}

	if info.IsDir() {
		return dirHasContent(path)
	}

	// Regular file exists
	return true, nil
}

func symlinkTargetExists(path string) (bool, error) {
	realPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	info, err := os.Stat(realPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	if info.IsDir() {
		return dirHasContent(realPath)
	}

	// Symlink to file
	return true, nil
}

func dirHasContent(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) > 0, nil
}

func run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("%w: %s", err, exitErr.Stderr)
		}
		return nil, err
	}
	return output, nil
}

// resolveHome expands a leading ~ in the path to the user's home directory.
// If the path doesn't start with ~, it is returned unchanged.
func resolveHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get user home dir: %w", err)
	}

	if path == "~" {
		return home, nil
	}

	// Handle ~/... pattern
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}

	// ~something without / is not supported (e.g., ~user)
	return path, nil
}

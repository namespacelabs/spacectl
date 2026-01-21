//go:generate moq -out mount_mock.go . Executor
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/namespacelabs/space/internal/cache/mode"
)

const (
	privateNamespaceDir = ".ns"
	metadataFilename    = "cache-metadata.json"
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
	DestructiveMode bool              `json:"destructive_mode,"`
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

	// Write metadata
	if err := m.writeMetadata(ctx, &result); err != nil {
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

	plan, err := modes.Plan(ctx, mode.PlanRequest{})
	if err != nil {
		return err
	}

	for modeName, p := range plan {
		for _, path := range p.MountPaths {
			mount, err := m.mountPath(ctx, modeName, path)
			if err != nil {
				return fmt.Errorf("mounting mode path %q: %w", path, err)
			}
			result.Output.Mounts = append(result.Output.Mounts, mount)
		}

		for k, v := range p.AddEnvs {
			if result.Output.AddEnvs == nil {
				result.Output.AddEnvs = make(map[string]string)
			}
			result.Output.AddEnvs[k] = v
		}

		for _, path := range p.RemovePaths {
			if err := m.removePath(ctx, path, result); err != nil {
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

func (m Mounter) writeMetadata(ctx context.Context, result *MountResponse) error {
	metadataPath := filepath.Join(m.CacheRoot, privateNamespaceDir, metadataFilename)

	if !m.DestructiveMode {
		slog.Debug("dry-run: would write cache metadata", slog.String("path", metadataPath))
		return nil
	}

	metadata := CacheMetadata{
		Version:     1,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		UserRequest: make(map[string]CacheMetadataEntry, len(result.Output.Mounts)),
	}

	for _, mount := range result.Output.Mounts {
		var cacheFramework *string
		if mount.Mode != "" {
			cacheFramework = &mount.Mode
		}

		metadata.UserRequest[mount.CachePath] = CacheMetadataEntry{
			CacheFramework: cacheFramework,
			MountTarget:    []string{mount.MountPath},
			Source:         "space",
		}
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	metadataDir := filepath.Dir(metadataPath)
	slog.Debug("creating metadata directory", slog.String("path", metadataDir))

	if err := m.Exec.MkdirAll(metadataDir, 0o755); err != nil {
		return fmt.Errorf("creating metadata directory: %w", err)
	}

	slog.Debug("writing cache metadata", slog.String("path", metadataPath))

	if err := m.Exec.WriteFile(metadataPath, data, 0o644); err != nil {
		return fmt.Errorf("writing metadata file: %w", err)
	}

	return nil
}

func (m Mounter) removePath(ctx context.Context, path string, result *MountResponse) error {
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
	mountPathEmpty, err := isEmptyDir(to)
	if err != nil {
		return fmt.Errorf("checking mount path content: %w", err)
	}
	if !mountPathEmpty {
		slog.Debug("mount path will be overwritten", slog.String("path", to))
	}

	slog.Debug("mounting path", slog.String("from", from), slog.String("to", to))

	// create cache path, this is noop if it already exists
	if err := os.MkdirAll(from, 0o755); err != nil {
		return fmt.Errorf("creating from path %q: %w", from, err)
	}

	// os specific mount logic
	return mount(ctx, from, to)
}

func (e DefaultExecutor) RemoveAll(name string) error {
	_, err := run(context.Background(), "sudo", "rm", "-rf", name)
	return err
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

func (e DefaultExecutor) DiskUsage(ctx context.Context, path string) (DiskUsage, error) {
	// TODO: make this more portable across different operating systems
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

func isEmptyDir(name string) (bool, error) {
	files, err := os.ReadDir(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	return len(files) == 0, nil
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

// ancestors returns all ancestor directories of the given path, from root to the path itself.
func ancestors(path string) []string {
	var result []string
	for path != "/" && path != "." {
		result = append(result, path)
		path = filepath.Dir(path)
	}

	// Reverse to get root-to-leaf order
	for i := len(result)/2 - 1; i >= 0; i-- {
		opp := len(result) - 1 - i
		result[i], result[opp] = result[opp], result[i]
	}

	return result
}

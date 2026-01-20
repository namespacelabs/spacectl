package cache_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namespacelabs/space/internal/cache"
	"github.com/namespacelabs/space/internal/cache/mode"
)

func TestMountRequest_EnabledModes(t *testing.T) {
	t.Run("manual modes only", func(t *testing.T) {
		req := cache.MountRequest{
			ManualModes: []string{"apt", "go"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.DefaultModes())
		require.NoError(t, err)
		require.Len(t, modes, 2)
		require.ElementsMatch(t, []string{"apt", "go"}, modes.Names())
	})

	t.Run("detect specific modes - all detected", func(t *testing.T) {
		req := cache.MountRequest{
			DetectModes: []string{"apt", "go"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
		})
		require.NoError(t, err)
		require.Len(t, modes, 2)
		require.ElementsMatch(t, []string{"apt", "go"}, modes.Names())
	})

	t.Run("detect specific modes - partially detected", func(t *testing.T) {
		req := cache.MountRequest{
			DetectModes: []string{"apt", "go"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "golangci-lint" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.NoError(t, err)
		require.Len(t, modes, 1)
		require.Equal(t, []string{"apt"}, modes.Names())
	})

	t.Run("detect specific modes - none detected", func(t *testing.T) {
		req := cache.MountRequest{
			DetectModes: []string{"apt", "go"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.NoError(t, err)
		require.Empty(t, modes)
	})

	t.Run("detect all modes", func(t *testing.T) {
		req := cache.MountRequest{
			DetectAllModes: true,
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "golangci-lint" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.NoError(t, err)
		require.Len(t, modes, 2)
		require.ElementsMatch(t, []string{"apt", "go"}, modes.Names())
	})

	t.Run("manual and detect combined", func(t *testing.T) {
		req := cache.MountRequest{
			ManualModes: []string{"apt"},
			DetectModes: []string{"go", "golangci-lint"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "golangci-lint" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.NoError(t, err)
		require.Len(t, modes, 2)
		require.ElementsMatch(t, []string{"apt", "go"}, modes.Names())
	})

	t.Run("empty enabled modes", func(t *testing.T) {
		req := cache.MountRequest{}

		_, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.Error(t, err)
	})

	t.Run("invalid manual mode", func(t *testing.T) {
		req := cache.MountRequest{
			ManualModes: []string{"invalid-mode"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "unknown mode: invalid-mode")
		require.Nil(t, modes)
	})

	t.Run("invalid detect mode", func(t *testing.T) {
		req := cache.MountRequest{
			DetectModes: []string{"invalid-mode"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "unknown mode: invalid-mode")
		require.Nil(t, modes)
	})

	t.Run("mixed valid and invalid modes", func(t *testing.T) {
		req := cache.MountRequest{
			ManualModes: []string{"apt", "invalid-mode"},
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "unknown mode: invalid-mode")
		require.Nil(t, modes)
	})

	t.Run("detection error propagates", func(t *testing.T) {
		req := cache.MountRequest{
			DetectAllModes: true,
		}

		modes, err := req.EnabledModes(t.Context(), mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "apt" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc: func() string { return "go" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
					return false, fmt.Errorf("detection failed")
				},
			},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "detecting go")
		require.ErrorContains(t, err, "detection failed")
		require.Nil(t, modes)
	})
}

func TestMount(t *testing.T) {
	t.Run("mount with manual modes", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.NoError(t, err)

		require.Equal(t, []string{"apt"}, result.Input.Modes)
		mountCalls := exec.MountCalls()
		require.Len(t, mountCalls, 1)
		require.Equal(t, filepath.Join(cacheRoot, mountPath), mountCalls[0].From)
		require.Equal(t, mountPath, mountCalls[0].To)

		// Verify Results contains the mount
		mounts := filterMounts(result.Output.Mounts)
		require.Len(t, mounts, 1)
		require.Equal(t, "apt", mounts[0].Mode)
		require.Equal(t, filepath.Join(cacheRoot, mountPath), mounts[0].CachePath)
		require.Equal(t, mountPath, mounts[0].MountPath)
	})

	t.Run("mount with detected modes", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return true, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath},
						}, nil
					},
				},
				&mode.ModeProviderMock{
					NameFunc:   func() string { return "go" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			DetectAllModes: true,
		})
		require.NoError(t, err)

		require.Equal(t, []string{"apt"}, result.Input.Modes)
		require.Len(t, exec.MountCalls(), 1)
	})

	t.Run("mount with manual paths", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		cachePath := filepath.Join(cacheRoot, mountPath)
		require.NoError(t, os.MkdirAll(cachePath, 0o755))

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				if name == cachePath {
					return nil, nil
				}
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes:           mode.Modes{},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualPaths: []string{mountPath},
		})
		require.NoError(t, err)

		require.Equal(t, []string{mountPath}, result.Input.Paths)
		mountCalls := exec.MountCalls()
		require.Len(t, mountCalls, 1)
		require.Equal(t, cachePath, mountCalls[0].From)
		require.Equal(t, mountPath, mountCalls[0].To)

		// Verify cache hit in Results
		mounts := filterMounts(result.Output.Mounts)
		require.Len(t, mounts, 1)
		require.True(t, mounts[0].CacheHit)
	})

	t.Run("cache hit vs miss tracking", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath1 := t.TempDir()
		mountPath2 := t.TempDir()

		cachePath1 := filepath.Join(cacheRoot, mountPath1)
		require.NoError(t, os.MkdirAll(cachePath1, 0o755))

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				if name == cachePath1 {
					return nil, nil
				}
				// cachePath2 doesn't exist
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath1, mountPath2},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.NoError(t, err)

		// Verify cache hits and misses in Results
		mounts := filterMounts(result.Output.Mounts)
		require.Len(t, mounts, 2)

		// Find hit and miss by mount path
		var hit, miss cache.MountResult
		for _, m := range mounts {
			switch m.MountPath {
			case mountPath1:
				hit = m
			case mountPath2:
				miss = m
			}
		}
		require.True(t, hit.CacheHit)
		require.False(t, miss.CacheHit)
	})

	t.Run("dry run does not mount", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: false, // dry run mode (default)
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.NoError(t, err)

		require.Empty(t, exec.MountCalls())
		require.Equal(t, []string{"apt"}, result.Input.Modes)

		// Verify Results still populated for dry run
		mounts := filterMounts(result.Output.Mounts)
		require.Len(t, mounts, 1)
	})

	t.Run("environment variables from modes", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			CacheRoot: cacheRoot,
			Exec:      exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "go" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath},
							AddEnvs: map[string]string{
								"GOMODCACHE": "/go/pkg/mod",
								"GOCACHE":    "/go/cache",
							},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"go"},
		})
		require.NoError(t, err)

		require.Len(t, result.Output.AddEnvs, 2)
		require.Equal(t, "/go/pkg/mod", result.Output.AddEnvs["GOMODCACHE"])
		require.Equal(t, "/go/cache", result.Output.AddEnvs["GOCACHE"])
	})

	t.Run("disk usage included in output", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{Total: "100G", Used: "50G"}, nil
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes:           mode.Modes{},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualPaths: []string{mountPath},
		})
		require.NoError(t, err)

		require.NotNil(t, result.Output.DiskUsage)
		require.Equal(t, "100G", result.Output.DiskUsage.Total)
		require.Equal(t, "50G", result.Output.DiskUsage.Used)
	})

	t.Run("disk usage error is suppressed", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("df command failed")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes:           mode.Modes{},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualPaths: []string{mountPath},
		})
		require.NoError(t, err)
		require.Nil(t, result.Output.DiskUsage)
	})

	t.Run("remove paths from modes", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			CacheRoot: cacheRoot,
			Exec:      exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths:  []string{mountPath},
							RemovePaths: []string{"/var/lib/apt/lists"},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.NoError(t, err)

		require.Equal(t, []string{"/var/lib/apt/lists"}, result.Output.RemovedPaths)
	})

	t.Run("remove paths in destructive mode", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()
		removePath := "/var/lib/apt/lists"

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			RemoveAllFunc: func(name string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths:  []string{mountPath},
							RemovePaths: []string{removePath},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.NoError(t, err)

		require.Equal(t, []string{removePath}, result.Output.RemovedPaths)
		removeCalls := exec.RemoveAllCalls()
		require.Len(t, removeCalls, 1)
		require.Equal(t, removePath, removeCalls[0].Name)
	})

	t.Run("dry-run does not remove paths", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			RemoveAllFunc: func(name string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: false,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths:  []string{mountPath},
							RemovePaths: []string{"/var/lib/apt/lists"},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.NoError(t, err)

		require.Equal(t, []string{"/var/lib/apt/lists"}, result.Output.RemovedPaths)
		require.Empty(t, exec.RemoveAllCalls())
	})

	t.Run("remove paths with multiple paths", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()
		removePaths := []string{"/var/lib/apt/lists", "/tmp/cache", "/var/cache/apt"}

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			RemoveAllFunc: func(name string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths:  []string{mountPath},
							RemovePaths: removePaths,
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.NoError(t, err)

		require.ElementsMatch(t, removePaths, result.Output.RemovedPaths)
		removeCalls := exec.RemoveAllCalls()
		require.Len(t, removeCalls, 3)
	})

	t.Run("remove error propagates", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			RemoveAllFunc: func(name string) error {
				return fmt.Errorf("remove failed")
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths:  []string{mountPath},
							RemovePaths: []string{"/var/lib/apt/lists"},
						}, nil
					},
				},
			},
		}

		_, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "remove failed")
	})

	t.Run("multiple modes combined", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath1 := t.TempDir()
		mountPath2 := t.TempDir()

		exec := &cache.ExecutorMock{
			MountFunc: func(ctx context.Context, from, to string) error {
				return nil
			},
			StatFunc: func(name string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			MkdirAllFunc: func(path string, perm os.FileMode) error {
				return nil
			},
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
				return nil
			},
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}
		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath1},
							AddEnvs:    map[string]string{"APT_VAR": "value1"},
						}, nil
					},
				},
				&mode.ModeProviderMock{
					NameFunc: func() string { return "go" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath2},
							AddEnvs:    map[string]string{"GOMODCACHE": "/go/pkg/mod"},
						}, nil
					},
				},
			},
		}

		result, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt", "go"},
		})
		require.NoError(t, err)

		require.ElementsMatch(t, []string{"apt", "go"}, result.Input.Modes)
		require.Len(t, exec.MountCalls(), 2)
		require.Len(t, result.Output.AddEnvs, 2)
	})

	t.Run("mount error propagates", func(t *testing.T) {
		cacheRoot := t.TempDir()
		mountPath := t.TempDir()

		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec: &cache.ExecutorMock{
				MountFunc: func(ctx context.Context, from, to string) error {
					return fmt.Errorf("mount failed")
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
				MkdirAllFunc: func(path string, perm os.FileMode) error {
					return nil
				},
				WriteFileFunc: func(name string, data []byte, perm os.FileMode) error {
					return nil
				},
			},
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc: func() string { return "apt" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
						return false, nil
					},
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{
							MountPaths: []string{mountPath},
						}, nil
					},
				},
			},
		}

		_, err := m.Mount(t.Context(), cache.MountRequest{
			ManualModes: []string{"apt"},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "mount failed")
	})

	t.Run("tilde path expansion", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		cacheRoot := t.TempDir()
		exec := &cache.ExecutorMock{
			MountFunc:     func(ctx context.Context, from, to string) error { return nil },
			StatFunc:      func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist },
			MkdirAllFunc:  func(path string, perm os.FileMode) error { return nil },
			WriteFileFunc: func(name string, data []byte, perm os.FileMode) error { return nil },
			DiskUsageFunc: func(ctx context.Context, path string) (cache.DiskUsage, error) {
				return cache.DiskUsage{}, fmt.Errorf("not implemented")
			},
		}

		m := cache.Mounter{
			DestructiveMode: true,
			CacheRoot:       cacheRoot,
			Exec:            exec,
			Modes: mode.Modes{
				&mode.ModeProviderMock{
					NameFunc:   func() string { return "test" },
					DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
					PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
						return mode.PlanResult{MountPaths: []string{"~/.cache/test"}}, nil
					},
				},
			},
		}

		_, err = m.Mount(t.Context(), cache.MountRequest{ManualModes: []string{"test"}})
		require.NoError(t, err)

		mountCalls := exec.MountCalls()
		require.Len(t, mountCalls, 1)
		require.Equal(t, filepath.Join(cacheRoot, homeDir, ".cache/test"), mountCalls[0].From)
		require.Equal(t, filepath.Join(homeDir, ".cache/test"), mountCalls[0].To)
	})
}

func filterMounts(mounts []cache.MountResult) []cache.MountResult {
	var result []cache.MountResult
	for _, m := range mounts {
		if m.MountPath != "" {
			result = append(result, m)
		}
	}
	return result
}

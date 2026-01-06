package mode_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namespacelabs/space/internal/cache/mode"
)

func ExampleModes() {
	detected, err := mode.DefaultModes().Detect(context.Background(), mode.DetectRequest{})
	if err != nil {
		panic(err)
	}

	result, err := detected.Plan(context.Background(), mode.PlanRequest{})
	if err != nil {
		panic(err)
	}

	err = json.NewEncoder(os.Stdout).Encode(result)
	if err != nil {
		panic(err)
	}
}

func TestModes_Available(t *testing.T) {
	modes := mode.Modes{
		mode.AptProvider{},
		mode.GolangCILintProvider{},
	}

	require.ElementsMatch(t, modes.Names(), []string{"apt", "golangci-lint"})
}

func TestModes_Filter(t *testing.T) {
	t.Run("filter single valid mode", func(t *testing.T) {
		filtered, err := mode.DefaultModes().Filter([]string{"apt"})
		require.NoError(t, err)
		require.Len(t, filtered, 1)
		require.Equal(t, "apt", filtered[0].Name())
	})

	t.Run("filter multiple valid modes", func(t *testing.T) {
		filtered, err := mode.DefaultModes().Filter([]string{"apt", "golangci-lint"})
		require.NoError(t, err)
		require.Len(t, filtered, 2)
		require.ElementsMatch(t, []string{"apt", "golangci-lint"}, []string{filtered[0].Name(), filtered[1].Name()})
	})

	t.Run("unknown mode returns error", func(t *testing.T) {
		filtered, err := mode.DefaultModes().Filter([]string{"unknown-mode"})
		require.Error(t, err)
		require.ErrorContains(t, err, "unknown mode: unknown-mode")
		require.Nil(t, filtered)
	})

	t.Run("mixed valid and invalid modes returns error", func(t *testing.T) {
		filtered, err := mode.DefaultModes().Filter([]string{"apt", "invalid"})
		require.Error(t, err)
		require.ErrorContains(t, err, "unknown mode: invalid")
		require.Nil(t, filtered)
	})
}

func TestModes_Detect(t *testing.T) {
	t.Run("empty modes returns empty", func(t *testing.T) {
		var modes mode.Modes
		detected, err := modes.Detect(t.Context(), mode.DetectRequest{})
		require.NoError(t, err)
		require.Empty(t, detected)
	})

	t.Run("all modes detected", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode1" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode2" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode3" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
		}
		detected, err := modes.Detect(t.Context(), mode.DetectRequest{})
		require.NoError(t, err)
		require.Len(t, detected, 3)
	})

	t.Run("no modes detected", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode1" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode2" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
		}
		detected, err := modes.Detect(t.Context(), mode.DetectRequest{})
		require.NoError(t, err)
		require.Empty(t, detected)
	})

	t.Run("some modes detected", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode1" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode2" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return false, nil },
			},
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode3" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
		}
		detected, err := modes.Detect(t.Context(), mode.DetectRequest{})
		require.NoError(t, err)
		require.Len(t, detected, 2)
		require.ElementsMatch(t, []string{"mode1", "mode3"}, []string{detected[0].Name(), detected[1].Name()})
	})

	t.Run("detection error returns error", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc:   func() string { return "mode1" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) { return true, nil },
			},
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode2" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
					return false, fmt.Errorf("detection failed")
				},
			},
		}
		detected, err := modes.Detect(t.Context(), mode.DetectRequest{})
		require.Error(t, err)
		require.ErrorContains(t, err, "detecting mode2")
		require.ErrorContains(t, err, "detection failed")
		require.Nil(t, detected)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode1" },
				DetectFunc: func(ctx context.Context, req mode.DetectRequest) (bool, error) {
					return false, ctx.Err()
				},
			},
		}
		detected, err := modes.Detect(ctx, mode.DetectRequest{})
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, detected)
	})
}

func TestModes_Plan(t *testing.T) {
	t.Run("empty modes returns empty map", func(t *testing.T) {
		var modes mode.Modes
		plans, err := modes.Plan(t.Context(), mode.PlanRequest{})
		require.NoError(t, err)
		require.Empty(t, plans)
	})

	t.Run("plans all modes", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode1" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{MountPaths: []string{"/cache1"}}, nil
				},
			},
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode2" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{MountPaths: []string{"/cache2"}}, nil
				},
			},
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode3" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{MountPaths: []string{"/cache3"}}, nil
				},
			},
		}
		plans, err := modes.Plan(t.Context(), mode.PlanRequest{})
		require.NoError(t, err)
		require.Len(t, plans, 3)
		require.Equal(t, []string{"/cache1"}, plans["mode1"].MountPaths)
		require.Equal(t, []string{"/cache2"}, plans["mode2"].MountPaths)
		require.Equal(t, []string{"/cache3"}, plans["mode3"].MountPaths)
	})

	t.Run("planning error returns error", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode1" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{MountPaths: []string{"/cache1"}}, nil
				},
			},
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode2" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{}, fmt.Errorf("planning failed")
				},
			},
		}
		plans, err := modes.Plan(t.Context(), mode.PlanRequest{})
		require.Error(t, err)
		require.ErrorContains(t, err, "planning mode2")
		require.ErrorContains(t, err, "planning failed")
		require.Nil(t, plans)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode1" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{}, ctx.Err()
				},
			},
		}
		plans, err := modes.Plan(ctx, mode.PlanRequest{})
		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		require.Nil(t, plans)
	})

	t.Run("collects all plan result fields", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode1" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{
						AddEnvs:     map[string]string{"KEY1": "value1"},
						MountPaths:  []string{"/cache1", "/cache2"},
						RemovePaths: []string{"/remove1"},
					}, nil
				},
			},
		}
		plans, err := modes.Plan(t.Context(), mode.PlanRequest{})
		require.NoError(t, err)
		require.Len(t, plans, 1)
		require.Equal(t, map[string]string{"KEY1": "value1"}, plans["mode1"].AddEnvs)
		require.Equal(t, []string{"/cache1", "/cache2"}, plans["mode1"].MountPaths)
		require.Equal(t, []string{"/remove1"}, plans["mode1"].RemovePaths)
	})

	t.Run("multiple modes with different results", func(t *testing.T) {
		modes := mode.Modes{
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode1" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{
						AddEnvs:    map[string]string{"KEY1": "value1"},
						MountPaths: []string{"/cache1"},
					}, nil
				},
			},
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode2" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{
						RemovePaths: []string{"/remove1", "/remove2"},
					}, nil
				},
			},
			&mode.ModeProviderMock{
				NameFunc: func() string { return "mode3" },
				PlanFunc: func(ctx context.Context, req mode.PlanRequest) (mode.PlanResult, error) {
					return mode.PlanResult{
						MountPaths: []string{"/cache3"},
					}, nil
				},
			},
		}
		plans, err := modes.Plan(t.Context(), mode.PlanRequest{})
		require.NoError(t, err)
		require.Len(t, plans, 3)
		require.Equal(t, map[string]string{"KEY1": "value1"}, plans["mode1"].AddEnvs)
		require.Equal(t, []string{"/cache1"}, plans["mode1"].MountPaths)
		require.Empty(t, plans["mode2"].MountPaths)
		require.Equal(t, []string{"/remove1", "/remove2"}, plans["mode2"].RemovePaths)
		require.Equal(t, []string{"/cache3"}, plans["mode3"].MountPaths)
	})
}

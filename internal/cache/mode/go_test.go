package mode_test

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namespacelabs/space/internal/cache/mode"
)

func TestGoProvider_Detect(t *testing.T) {
	t.Run("detected", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/go", nil
				},
			},
		}

		p := mode.GoProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.GoProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestGoProvider_Plan(t *testing.T) {
	t.Run("cache paths extracted", func(t *testing.T) {
		goEnvOutput := []byte(`{"GOCACHE":"/home/user/.cache/go-build","GOMODCACHE":"/home/user/go/pkg/mod"}`)

		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return goEnvOutput, nil
				},
			},
		}

		p := mode.GoProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, 2, len(result.MountPaths))
		require.Equal(t, "/home/user/.cache/go-build", result.MountPaths[0])
		require.Equal(t, "/home/user/go/pkg/mod", result.MountPaths[1])
	})
}

func TestGolangCILintProvider_Detect(t *testing.T) {
	t.Run("detected", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/golangci-lint", nil
				},
			},
		}

		p := mode.GolangCILintProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.GolangCILintProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestGolangCILintProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		cacheStatusOutput := []byte(`
			Dir: /home/user/.cache/golangci-lint
			Size: 123MB
		`)

		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return cacheStatusOutput, nil
				},
			},
		}

		p := mode.GolangCILintProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, 1, len(result.MountPaths))
		require.Equal(t, "/home/user/.cache/golangci-lint", result.MountPaths[0])
	})
}

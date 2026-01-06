package mode_test

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namespacelabs/space/internal/cache/mode"
)

func TestAptProvider_Detect(t *testing.T) {
	t.Run("detected", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/apt-config", nil
				},
			},
		}

		p := mode.AptProvider{}
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

		p := mode.AptProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestAptProvider_Plan(t *testing.T) {
	defaultAptConfig := []byte(`
		Dir::Cache "var/cache/apt";
		Dir::Cache::archives "archives/";
		Dir::Etc "etc/apt";
		Dir::Etc::parts "apt.conf.d";
	`)

	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return defaultAptConfig, nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.AptProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, 1, len(result.MountPaths))
		require.Equal(t, 0, len(result.RemovePaths))
		require.Equal(t, "/var/cache/apt/archives/", result.MountPaths[0])
	})

	t.Run("docker-clean removed", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return defaultAptConfig, nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, nil // no error means file exists
				},
			},
		}

		p := mode.AptProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, 1, len(result.MountPaths))
		require.Equal(t, 1, len(result.RemovePaths))
		require.Equal(t, "/etc/apt/apt.conf.d/docker-clean", result.RemovePaths[0])
	})
}

package mode_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namespacelabs/spacectl/internal/cache/mode"
)

// AptProvider tests

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

// BrewProvider tests

func TestBrewProvider_Detect(t *testing.T) {
	t.Run("detected when binary and Brewfile exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/brew", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "Brewfile", name)
					return nil, nil
				},
			},
		}

		p := mode.BrewProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.BrewProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when Brewfile missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/brew", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.BrewProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestBrewProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte("/Users/user/Library/Caches/Homebrew\n"), nil
				},
			},
		}

		p := mode.BrewProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/Users/user/Library/Caches/Homebrew"}, result.MountPaths)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(""), nil
				},
			},
		}

		p := mode.BrewProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

// BunProvider tests

func TestBunProvider_Detect(t *testing.T) {
	t.Run("detected when binary and lock file exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/bun", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "bun.lock", name)
					return nil, nil
				},
			},
		}

		p := mode.BunProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.BunProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when lock file missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/bun", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.BunProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestBunProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte("/home/user/.bun/install/cache\n"), nil
				},
			},
		}

		p := mode.BunProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.bun/install/cache"}, result.MountPaths)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(""), nil
				},
			},
		}

		p := mode.BunProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

// CocoapodsProvider tests

func TestCocoapodsProvider_Detect(t *testing.T) {
	t.Run("detected when binary and Podfile exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/pod", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "Podfile", name)
					return nil, nil
				},
			},
		}

		p := mode.CocoapodsProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.CocoapodsProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when Podfile missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/pod", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.CocoapodsProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestCocoapodsProvider_Plan(t *testing.T) {
	t.Run("returns mount paths", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.CocoapodsProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Len(t, result.MountPaths, 2)
		require.Equal(t, "./Pods", result.MountPaths[0])
		require.Equal(t, "~/Library/Caches/CocoaPods", result.MountPaths[1])
	})
}

// ComposerProvider tests

func TestComposerProvider_Detect(t *testing.T) {
	t.Run("detected when binary and composer.json exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/composer", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "composer.json", name)
					return nil, nil
				},
			},
		}

		p := mode.ComposerProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.ComposerProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when composer.json missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/composer", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.ComposerProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestComposerProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte("/home/user/.composer/cache/files\n"), nil
				},
			},
		}

		p := mode.ComposerProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.composer/cache/files"}, result.MountPaths)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(""), nil
				},
			},
		}

		p := mode.ComposerProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

// DenoProvider tests

func TestDenoProvider_Detect(t *testing.T) {
	t.Run("detected when binary and lock file exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/deno", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "deno.lock", name)
					return nil, nil
				},
			},
		}

		p := mode.DenoProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.DenoProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when lock file missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/deno", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.DenoProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestDenoProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(`{"denoDir":"/home/user/.cache/deno"}`), nil
				},
			},
		}

		p := mode.DenoProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.cache/deno"}, result.MountPaths)
	})

	t.Run("missing denoDir returns error", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(`{"otherKey":"value"}`), nil
				},
			},
		}

		p := mode.DenoProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "denoDir not found")
	})
}

// GoProvider tests

func TestGoProvider_Detect(t *testing.T) {
	t.Run("detected when binary and go.mod exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/go", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "go.mod" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.GoProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("detected when binary and go.work exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/go", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "go.work" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.GoProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
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

	t.Run("not detected when go.mod and go.work missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/go", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
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

// GolangCILintProvider tests

func TestGolangCILintProvider_Detect(t *testing.T) {
	t.Run("detected when binary and .golangci.yml exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/golangci-lint", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == ".golangci.yml" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.GolangCILintProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("detected when binary and .golangci.yaml exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/golangci-lint", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == ".golangci.yaml" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.GolangCILintProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
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

	t.Run("not detected when config files missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/golangci-lint", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
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

	t.Run("uses default path when dir not in output", func(t *testing.T) {
		cacheStatusOutput := []byte(`
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
		require.Equal(t, []string{"~/.cache/golangci-lint"}, result.MountPaths)
	})
}

// GradleProvider tests

func TestGradleProvider_Detect(t *testing.T) {
	t.Run("detected when binary and gradlew exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/gradle", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "gradlew" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.GradleProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("detected when binary and build.gradle exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/gradle", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "build.gradle" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.GradleProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.GradleProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when gradlew and build.gradle missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/gradle", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.GradleProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestGradleProvider_Plan(t *testing.T) {
	t.Run("returns mount paths", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.GradleProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Len(t, result.MountPaths, 2)
		require.Equal(t, "~/.gradle/caches", result.MountPaths[0])
		require.Equal(t, "~/.gradle/wrapper", result.MountPaths[1])
	})
}

// MavenProvider tests

func TestMavenProvider_Detect(t *testing.T) {
	t.Run("detected when binary and pom.xml exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/mvn", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "pom.xml", name)
					return nil, nil
				},
			},
		}

		p := mode.MavenProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.MavenProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when pom.xml missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/mvn", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.MavenProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestMavenProvider_Plan(t *testing.T) {
	t.Run("returns mount path", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.MavenProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"~/.m2/repository"}, result.MountPaths)
	})
}

// MiseProvider tests

func TestMiseProvider_Detect(t *testing.T) {
	t.Run("detected when binary and mise.toml exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/mise", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "mise.toml" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.MiseProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("detected when binary and .tool-versions exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/mise", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == ".tool-versions" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.MiseProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.MiseProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when config files missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/mise", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.MiseProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestMiseProvider_Plan(t *testing.T) {
	t.Run("uses MISE_DATA_DIR when set", func(t *testing.T) {
		t.Setenv("MISE_DATA_DIR", "/custom/mise/dir")

		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.MiseProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/custom/mise/dir"}, result.MountPaths)
	})

	t.Run("uses XDG_DATA_HOME when MISE_DATA_DIR not set", func(t *testing.T) {
		t.Setenv("MISE_DATA_DIR", "")
		t.Setenv("XDG_DATA_HOME", "/custom/xdg/data")

		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.MiseProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{filepath.Join("/custom/xdg/data", "mise")}, result.MountPaths)
	})

	t.Run("uses default path when no env vars set", func(t *testing.T) {
		t.Setenv("MISE_DATA_DIR", "")
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("LOCALAPPDATA", "")

		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.MiseProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Len(t, result.MountPaths, 1)
		require.Contains(t, result.MountPaths[0], filepath.Join(".local", "share", "mise"))
	})
}

// NixProvider tests

func TestNixProvider_Detect(t *testing.T) {
	t.Run("detected when binary and flake.nix exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/nix/var/nix/profiles/default/bin/nix", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "flake.nix" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.NixProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("detected when binary and shell.nix exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/nix/var/nix/profiles/default/bin/nix", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "shell.nix" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.NixProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("detected when binary and default.nix exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/nix/var/nix/profiles/default/bin/nix", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					if name == "default.nix" {
						return nil, nil
					}
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.NixProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.NixProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when project files missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/nix/var/nix/profiles/default/bin/nix", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.NixProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestNixProvider_Plan(t *testing.T) {
	t.Run("returns mount paths", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.NixProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"~/.cache/nix", "/nix"}, result.MountPaths)
	})
}

// PlaywrightProvider tests

func TestPlaywrightProvider_Detect(t *testing.T) {
	t.Run("detected", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/playwright", nil
				},
			},
		}

		p := mode.PlaywrightProvider{}
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

		p := mode.PlaywrightProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestPlaywrightProvider_Plan(t *testing.T) {
	t.Run("uses PLAYWRIGHT_BROWSERS_PATH when set", func(t *testing.T) {
		t.Setenv("PLAYWRIGHT_BROWSERS_PATH", "/custom/playwright/path")

		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.PlaywrightProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/custom/playwright/path"}, result.MountPaths)
	})

	t.Run("uses default path when no env var set", func(t *testing.T) {
		t.Setenv("PLAYWRIGHT_BROWSERS_PATH", "")

		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.PlaywrightProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Len(t, result.MountPaths, 1)
		require.Contains(t, result.MountPaths[0], "ms-playwright")
	})
}

// PnpmProvider tests

func TestPnpmProvider_Detect(t *testing.T) {
	t.Run("detected when binary and pnpm-lock.yaml exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/pnpm", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "pnpm-lock.yaml", name)
					return nil, nil
				},
			},
		}

		p := mode.PnpmProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.PnpmProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when pnpm-lock.yaml missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/pnpm", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.PnpmProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestPnpmProvider_Plan(t *testing.T) {
	t.Run("cache path extracted with new version", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("9.7.0\n"), nil // version
					}
					return []byte("/home/user/.local/share/pnpm/store/v3\n"), nil
				},
			},
		}

		p := mode.PnpmProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.local/share/pnpm/store/v3"}, result.MountPaths)
		require.Equal(t, map[string]string{"npm_config_package_import_method": "copy"}, result.AddEnvs)
	})

	t.Run("old version extracts version from last line when warnings present", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						// pnpm < 9.7.0 prints warnings to stdout, version is on last line
						return []byte("\u2009WARN\u2009 some warning\n9.6.0\n"), nil
					}
					return []byte("\u2009WARN\u2009 some warning\n/home/user/.local/share/pnpm/store/v3\n"), nil
				},
			},
		}

		p := mode.PnpmProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.local/share/pnpm/store/v3"}, result.MountPaths)
	})

	t.Run("cache path extracted with old version drops single warning", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("9.6.0\n"), nil // old version
					}
					// pnpm uses thin spaces (\u2009) around WARN
					return []byte("\u2009WARN\u2009 some warning\n/home/user/.local/share/pnpm/store/v3\n"), nil
				},
			},
		}

		p := mode.PnpmProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.local/share/pnpm/store/v3"}, result.MountPaths)
	})

	t.Run("cache path extracted with old version drops multiple warnings", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("9.5.0\n"), nil // old version
					}
					// Multiple warning lines with thin spaces
					return []byte("\u2009WARN\u2009 deprecated package\n\u2009WARN\u2009 another warning\n/home/user/.local/share/pnpm/store/v3\n"), nil
				},
			},
		}

		p := mode.PnpmProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.local/share/pnpm/store/v3"}, result.MountPaths)
	})

	t.Run("new version does not filter warnings", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("9.7.0\n"), nil // new version
					}
					// New versions don't print warnings to stdout with --loglevel error
					return []byte("/home/user/.local/share/pnpm/store/v3\n"), nil
				},
			},
		}

		p := mode.PnpmProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.local/share/pnpm/store/v3"}, result.MountPaths)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("9.7.0\n"), nil
					}
					return []byte(""), nil
				},
			},
		}

		p := mode.PnpmProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

// PoetryProvider tests

func TestPoetryProvider_Detect(t *testing.T) {
	t.Run("detected when binary and poetry.lock exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/poetry", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "poetry.lock", name)
					return nil, nil
				},
			},
		}

		p := mode.PoetryProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.PoetryProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when poetry.lock missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/poetry", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.PoetryProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestPoetryProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte("/home/user/.cache/pypoetry\n"), nil
				},
			},
		}

		p := mode.PoetryProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.cache/pypoetry"}, result.MountPaths)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(""), nil
				},
			},
		}

		p := mode.PoetryProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

// PythonProvider tests

func TestPythonProvider_Detect(t *testing.T) {
	t.Run("detected when binary and requirements.txt exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/pip", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "requirements.txt", name)
					return nil, nil
				},
			},
		}

		p := mode.PythonProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.PythonProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when requirements.txt missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/pip", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.PythonProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestPythonProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte("/home/user/.cache/pip\n"), nil
				},
			},
		}

		p := mode.PythonProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.cache/pip"}, result.MountPaths)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(""), nil
				},
			},
		}

		p := mode.PythonProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

// RubyProvider tests

func TestRubyProvider_Detect(t *testing.T) {
	t.Run("detected when binary and Gemfile exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/bundle", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "Gemfile", name)
					return nil, nil
				},
			},
		}

		p := mode.RubyProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.RubyProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when Gemfile missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/bundle", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.RubyProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestRubyProvider_Plan(t *testing.T) {
	t.Run("returns vendor paths", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.RubyProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"./vendor/bundle", "./vendor/cache"}, result.MountPaths)
	})
}

// RustProvider tests

func TestRustProvider_Detect(t *testing.T) {
	t.Run("detected when binary and Cargo.toml exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/cargo", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "Cargo.toml", name)
					return nil, nil
				},
			},
		}

		p := mode.RustProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.RustProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when Cargo.toml missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/cargo", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.RustProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestRustProvider_Plan(t *testing.T) {
	t.Run("returns cargo and target paths", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.RustProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{
			"~/.cargo/registry",
			"~/.cargo/git",
			"./target",
			"~/.cargo/.global-cache",
		}, result.MountPaths)
	})
}

// SwiftPMProvider tests

func TestSwiftPMProvider_Detect(t *testing.T) {
	t.Run("detected when binary and Package.swift exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/swift", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "Package.swift", name)
					return nil, nil
				},
			},
		}

		p := mode.SwiftPMProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.SwiftPMProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when Package.swift missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/swift", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.SwiftPMProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestSwiftPMProvider_Plan(t *testing.T) {
	t.Run("returns all paths when xcode mode not enabled", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.SwiftPMProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{
			"./.build",
			"~/Library/Caches/org.swift.swiftpm",
			"~/Library/org.swift.swiftpm",
			"~/Library/Developer/Xcode/DerivedData/ModuleCache.noindex",
		}, result.MountPaths)
	})

	t.Run("excludes module cache when xcode mode enabled", func(t *testing.T) {
		req := mode.PlanRequest{
			EnabledModes: []string{"swiftpm", "xcode"},
			Exec:         &mode.ExecutorMock{},
		}

		p := mode.SwiftPMProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{
			"./.build",
			"~/Library/Caches/org.swift.swiftpm",
			"~/Library/org.swift.swiftpm",
		}, result.MountPaths)
	})
}

// UVProvider tests

func TestUVProvider_Detect(t *testing.T) {
	t.Run("detected when binary and uv.lock exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/uv", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "uv.lock", name)
					return nil, nil
				},
			},
		}

		p := mode.UVProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.UVProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when uv.lock missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/uv", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.UVProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestUVProvider_Plan(t *testing.T) {
	t.Run("cache path extracted", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte("/home/user/.cache/uv\n"), nil
				},
			},
		}

		p := mode.UVProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.cache/uv"}, result.MountPaths)
		require.Equal(t, map[string]string{"UV_LINK_MODE": "symlink"}, result.AddEnvs)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					return []byte(""), nil
				},
			},
		}

		p := mode.UVProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

// XcodeProvider tests

type mockDirEntry struct {
	name  string
	isDir bool
}

func (m mockDirEntry) Name() string               { return m.name }
func (m mockDirEntry) IsDir() bool                { return m.isDir }
func (m mockDirEntry) Type() os.FileMode          { return 0 }
func (m mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestXcodeProvider_Detect(t *testing.T) {
	t.Run("detected when binary and .xcodeproj exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/xcodebuild", nil
				},
				ReadDirFunc: func(name string) ([]os.DirEntry, error) {
					return []os.DirEntry{
						mockDirEntry{name: "MyApp.xcodeproj", isDir: true},
					}, nil
				},
			},
		}

		p := mode.XcodeProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.XcodeProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when no .xcodeproj exists", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/bin/xcodebuild", nil
				},
				ReadDirFunc: func(name string) ([]os.DirEntry, error) {
					return []os.DirEntry{
						mockDirEntry{name: "README.md", isDir: false},
						mockDirEntry{name: "src", isDir: true},
					}, nil
				},
			},
		}

		p := mode.XcodeProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestXcodeProvider_Plan(t *testing.T) {
	t.Run("returns cache path and env", func(t *testing.T) {
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{},
		}

		p := mode.XcodeProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"~/Library/Developer/Xcode/DerivedData/CompilationCache.noindex"}, result.MountPaths)
		require.Equal(t, map[string]string{"COMPILATION_CACHE_ENABLE_CACHING_DEFAULT": "YES"}, result.AddEnvs)
	})
}

// YarnProvider tests

func TestYarnProvider_Detect(t *testing.T) {
	t.Run("detected when binary and lock file exist", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/yarn", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					require.Equal(t, "yarn.lock", name)
					return nil, nil
				},
			},
		}

		p := mode.YarnProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.True(t, detected)
	})

	t.Run("not detected when binary missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "", exec.ErrNotFound
				},
			},
		}

		p := mode.YarnProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})

	t.Run("not detected when lock file missing", func(t *testing.T) {
		req := mode.DetectRequest{
			Exec: &mode.ExecutorMock{
				LookPathFunc: func(file string) (string, error) {
					return "/usr/local/bin/yarn", nil
				},
				StatFunc: func(name string) (os.FileInfo, error) {
					return nil, os.ErrNotExist
				},
			},
		}

		p := mode.YarnProvider{}
		detected, err := p.Detect(t.Context(), req)
		require.NoError(t, err)
		require.False(t, detected)
	})
}

func TestYarnProvider_Plan(t *testing.T) {
	t.Run("yarn v1 uses cache dir command", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("1.22.19\n"), nil
					}
					return []byte("/home/user/.cache/yarn/v6\n"), nil
				},
			},
		}

		p := mode.YarnProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.cache/yarn/v6"}, result.MountPaths)
	})

	t.Run("yarn v2+ uses config get cacheFolder command", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("4.0.2\n"), nil
					}
					return []byte("/home/user/.yarn/cache\n"), nil
				},
			},
		}

		p := mode.YarnProvider{}
		result, err := p.Plan(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, []string{"/home/user/.yarn/cache"}, result.MountPaths)
	})

	t.Run("empty cache dir returns error", func(t *testing.T) {
		callCount := 0
		req := mode.PlanRequest{
			Exec: &mode.ExecutorMock{
				OutputFunc: func(cmd *exec.Cmd) ([]byte, error) {
					callCount++
					if callCount == 1 {
						return []byte("1.22.19\n"), nil
					}
					return []byte(""), nil
				},
			},
		}

		p := mode.YarnProvider{}
		_, err := p.Plan(t.Context(), req)
		require.Error(t, err)
		require.Contains(t, err.Error(), "empty cache dir")
	})
}

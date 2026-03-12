package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestIntegration_CacheMount(t *testing.T) {
	binary := os.Getenv("INTEGRATION_SPACECTL_BIN")
	if binary == "" {
		t.Skip("set INTEGRATION_SPACECTL_BIN to run this integration test")
	}

	// Verifies that --path=a,b is split into two separate mount paths.
	// The --path flag must remain a StringSlice (not StringArray) so that
	// users and CI scripts can pass multiple paths in a single flag value.
	t.Run("comma-separated paths", func(t *testing.T) {
		t.Setenv("NSC_CACHE_PATH", t.TempDir())

		pathA := t.TempDir()
		pathB := t.TempDir()

		resp := runMount(t, binary,
			"--path="+pathA+","+pathB,
		)

		if len(resp.Output.Mounts) != 2 {
			t.Fatalf("--path=a,b was not split on comma: got %d mount(s), want 2", len(resp.Output.Mounts))
		}
	})

	// Verifies that --path can be passed multiple times.
	t.Run("repeated path flags", func(t *testing.T) {
		t.Setenv("NSC_CACHE_PATH", t.TempDir())

		pathA := t.TempDir()
		pathB := t.TempDir()

		resp := runMount(t, binary,
			"--path="+pathA,
			"--path="+pathB,
		)

		if len(resp.Output.Mounts) != 2 {
			t.Fatalf("repeated --path flags: got %d mount(s), want 2", len(resp.Output.Mounts))
		}
	})

	// Verifies that --mode=a,b is split into two separate modes.
	t.Run("comma-separated modes", func(t *testing.T) {
		t.Setenv("NSC_CACHE_PATH", t.TempDir())

		resp := runMount(t, binary,
			"--mode=go,ruby",
		)

		if len(resp.Input.Modes) != 2 {
			t.Fatalf("--mode=a,b was not split on comma: got modes %v, want 2", resp.Input.Modes)
		}
	})

	// Verifies that --mode can be passed multiple times.
	t.Run("repeated mode flags", func(t *testing.T) {
		t.Setenv("NSC_CACHE_PATH", t.TempDir())

		resp := runMount(t, binary,
			"--mode=go",
			"--mode=ruby",
		)

		if len(resp.Input.Modes) != 2 {
			t.Fatalf("repeated --mode flags: got modes %v, want 2", resp.Input.Modes)
		}
	})

	// Verifies that --detect=a,b is split into two separate detectors.
	// Without comma splitting, "go,ruby" would be treated as a single unknown
	// mode name and the command would fail with "unknown mode: go,ruby".
	t.Run("comma-separated detect", func(t *testing.T) {
		t.Setenv("NSC_CACHE_PATH", t.TempDir())

		resp := runMount(t, binary,
			"--detect=go,ruby",
			"--path="+t.TempDir(),
		)

		if resp.Output.Mounts == nil {
			t.Fatal("expected mounts in response")
		}
	})

	// Verifies that --detect can be passed multiple times.
	t.Run("repeated detect flags", func(t *testing.T) {
		t.Setenv("NSC_CACHE_PATH", t.TempDir())

		resp := runMount(t, binary,
			"--detect=go",
			"--detect=ruby",
			"--path="+t.TempDir(),
		)

		if resp.Output.Mounts == nil {
			t.Fatal("expected mounts in response")
		}
	})
}

type mountResponse struct {
	Input struct {
		Modes []string `json:"modes"`
		Paths []string `json:"paths"`
	} `json:"input"`
	Output struct {
		Mounts []struct {
			MountPath string `json:"mount_path"`
		} `json:"mounts"`
	} `json:"output"`
}

func runMount(t *testing.T, binary string, extraArgs ...string) mountResponse {
	t.Helper()

	args := append([]string{"cache", "mount", "--dry_run=true", "-o=json"}, extraArgs...)
	cmd := exec.Command(binary, args...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("spacectl %s failed: %s", strings.Join(args, " "), output)
	}

	var resp mountResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		t.Fatalf("failed to parse JSON output: %v\n%s", err, output)
	}
	return resp
}

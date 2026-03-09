package cmd_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestIntegration_Version(t *testing.T) {
	binary := os.Getenv("INTEGRATION_SPACECTL_BIN")
	if binary == "" {
		t.Skip("set INTEGRATION_SPACECTL_BIN to run this integration test")
	}

	// Verifies that version information baked in via ldflags is correctly
	// wired through to the JSON output. The CI build injects known values
	// so we can assert exact matches.
	t.Run("json output matches build info", func(t *testing.T) {
		resp := runVersion(t, binary)

		wantVersion := os.Getenv("INTEGRATION_SPACECTL_BIN_VERSION")
		wantCommit := os.Getenv("INTEGRATION_SPACECTL_BIN_COMMIT")
		wantDate := os.Getenv("INTEGRATION_SPACECTL_BIN_DATE")

		if wantVersion == "" || wantCommit == "" || wantDate == "" {
			t.Skip("set INTEGRATION_SPACECTL_BIN_VERSION, INTEGRATION_SPACECTL_BIN_COMMIT, and INTEGRATION_SPACECTL_BIN_DATE to run this test")
		}

		if resp.Version != wantVersion {
			t.Errorf("version = %q, want %q", resp.Version, wantVersion)
		}
		if resp.Commit != wantCommit {
			t.Errorf("commit = %q, want %q", resp.Commit, wantCommit)
		}
		if resp.Date != wantDate {
			t.Errorf("date = %q, want %q", resp.Date, wantDate)
		}
	})

	// Verifies that the plain text output includes the version identifier.
	t.Run("plain output contains version string", func(t *testing.T) {
		cmd := exec.Command(binary, "version")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("spacectl version failed: %s", output)
		}

		out := string(output)
		if !strings.Contains(out, "Spacectl CLI") {
			t.Errorf("plain output missing 'Spacectl CLI': got %q", out)
		}
	})
}

type versionResponse struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func runVersion(t *testing.T, binary string) versionResponse {
	t.Helper()

	args := []string{"version", "-o=json"}
	cmd := exec.Command(binary, args...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("spacectl %s failed: %s", strings.Join(args, " "), output)
	}

	var resp versionResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		t.Fatalf("failed to parse JSON output: %v\n%s", err, output)
	}
	return resp
}

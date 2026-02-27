package mode

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestXcodeDerivedDataHash_Integration validates our hash implementation against
// the actual DerivedData directory that Xcode creates. It requires macOS with
// Xcode installed.
//
// Run with: NSC_TEST_XCODE_HASH=1 go test -v -run TestXcodeDerivedDataHash_Integration ./internal/cache/mode/
func TestXcodeDerivedDataHash_Integration(t *testing.T) {
	if os.Getenv("NSC_TEST_XCODE_HASH") == "" {
		t.Skip("set NSC_TEST_XCODE_HASH=1 to run this integration test")
	}

	// Copy the minimal fixture to a temp dir so the absolute path is unique.
	tmpDir := t.TempDir()
	copyDir(t, "testdata", tmpDir)

	projectFile := "TestApp.xcodeproj"
	absPath := filepath.Join(tmpDir, projectFile)

	name := strings.TrimSuffix(projectFile, xcodeProjSuffix)
	computedHash := xcodeDerivedDataHash(absPath)
	computedSubfolder := name + "-" + computedHash

	// Run xcodebuild to trigger DerivedData directory creation.
	runXcodebuild(t, tmpDir, projectFile)

	// Scan ~/Library/Developer/Xcode/DerivedData/ for the actual subfolder.
	actualSubfolder := findDerivedDataSubfolder(t, name)

	t.Logf("Absolute path:      %s", absPath)
	t.Logf("Computed subfolder: %s", computedSubfolder)
	t.Logf("Actual subfolder:   %s", actualSubfolder)

	if computedSubfolder != actualSubfolder {
		t.Errorf("hash mismatch: computed %q, actual %q — Xcode's hash algorithm may have changed", computedSubfolder, actualSubfolder)
	}
}

func runXcodebuild(t *testing.T, dir, projectFile string) {
	t.Helper()

	projectPath := filepath.Join(dir, projectFile)

	// We only need xcodebuild to register the project in DerivedData.
	// -showBuildSettings is the lightest operation.
	args := []string{"-showBuildSettings", "-project", projectPath}

	cmd := exec.Command("xcodebuild", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("xcodebuild -showBuildSettings: %v\n%s", err, output)
	}
}

func findDerivedDataSubfolder(t *testing.T, projectName string) string {
	t.Helper()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home dir: %v", err)
	}
	ddDir := filepath.Join(homeDir, "Library", "Developer", "Xcode", "DerivedData")
	entries, err := os.ReadDir(ddDir)
	if err != nil {
		t.Fatalf("readdir %s: %v", ddDir, err)
	}

	prefix := projectName + "-"
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) && entry.IsDir() {
			return entry.Name()
		}
	}

	t.Fatalf("no DerivedData subfolder found for project %q in %s", projectName, ddDir)
	return ""
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copy %s -> %s: %v", src, dst, err)
	}
}

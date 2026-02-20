package mode

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestXcodeDerivedDataHash_Integration validates our hash implementation against
// the real xcodebuild -showBuildSettings output. It requires macOS with Xcode
// installed and a .xcodeproj or .xcworkspace in the current directory.
//
// Run with: NSC_TEST_XCODE_HASH=1 go test -run TestXcodeDerivedDataHash_Integration ./internal/cache/mode/
func TestXcodeDerivedDataHash_Integration(t *testing.T) {
	if os.Getenv("NSC_TEST_XCODE_HASH") == "" {
		t.Skip("set NSC_TEST_XCODE_HASH=1 to run this integration test")
	}

	// Find .xcworkspace or .xcodeproj in the current directory.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	var projectFile string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), xcodeWorkspaceSuffix) {
			projectFile = entry.Name()
			break
		}
	}
	if projectFile == "" {
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), xcodeProjSuffix) {
				projectFile = entry.Name()
				break
			}
		}
	}
	if projectFile == "" {
		t.Fatal("no .xcodeproj or .xcworkspace found in current directory")
	}

	// Compute expected hash from our algorithm.
	absPath, err := filepath.Abs(projectFile)
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	name := strings.TrimSuffix(strings.TrimSuffix(projectFile, xcodeWorkspaceSuffix), xcodeProjSuffix)
	computedHash := xcodeDerivedDataHash(absPath)
	computedSubfolder := name + "-" + computedHash

	// Get actual BUILD_DIR from xcodebuild.
	var args []string
	if strings.HasSuffix(projectFile, xcodeWorkspaceSuffix) {
		// Discover a scheme first.
		listCmd := exec.Command("xcodebuild", "-list", "-workspace", projectFile)
		listOutput, err := listCmd.Output()
		if err != nil {
			t.Fatalf("xcodebuild -list: %v", err)
		}
		scheme := parseFirstScheme(listOutput)
		if scheme == "" {
			t.Fatal("no scheme found in xcodebuild -list output")
		}
		args = []string{"-showBuildSettings", "-workspace", projectFile, "-scheme", scheme}
	} else {
		args = []string{"-showBuildSettings", "-project", projectFile}
	}

	cmd := exec.Command("xcodebuild", args...)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("xcodebuild -showBuildSettings: %v", err)
	}

	// Parse BUILD_DIR to extract the DerivedData subfolder.
	var buildDir string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, ok := strings.Cut(line, " = ")
		if ok && strings.TrimSpace(key) == "BUILD_DIR" {
			buildDir = strings.TrimSpace(value)
			break
		}
	}
	if buildDir == "" {
		t.Fatal("BUILD_DIR not found in xcodebuild -showBuildSettings output")
	}

	// Extract the subfolder: .../DerivedData/<subfolder>/Build/Products
	parts := strings.Split(buildDir, "/")
	var actualSubfolder string
	for i, p := range parts {
		if p == "DerivedData" && i+1 < len(parts) {
			actualSubfolder = parts[i+1]
			break
		}
	}
	if actualSubfolder == "" {
		t.Fatalf("could not extract DerivedData subfolder from BUILD_DIR: %s", buildDir)
	}

	t.Logf("Project file:       %s", projectFile)
	t.Logf("Absolute path:      %s", absPath)
	t.Logf("Computed subfolder: %s", computedSubfolder)
	t.Logf("Actual subfolder:   %s", actualSubfolder)

	if computedSubfolder != actualSubfolder {
		t.Errorf("hash mismatch: computed %q, actual %q — Xcode's hash algorithm may have changed", computedSubfolder, actualSubfolder)
	}
}

func parseFirstScheme(output []byte) string {
	var inSchemes bool
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "Schemes:" {
			inSchemes = true
			continue
		}
		if inSchemes && line != "" {
			return line
		}
	}
	return ""
}

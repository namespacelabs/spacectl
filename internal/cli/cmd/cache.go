package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/namespacelabs/space/internal/cache"
	"github.com/namespacelabs/space/internal/cache/mode"
)

const defaultCacheRootEnv = "NSC_CACHE_PATH"

func NewCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Take full advantage of Namespace volumes and caching infrastructure",
	}

	cmd.AddCommand(newCacheModesCmd())
	cmd.AddCommand(newCacheMountCmd())

	return cmd
}

func newCacheModesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modes",
		Short: "List available cache modes",
	}

	outputFlag := cmd.Flags().StringP("output", "o", "plain", "Output format: plain or json.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		modes := mode.DefaultModes()
		detected, err := modes.Detect(cmd.Context(), mode.DetectRequest{})
		if err != nil {
			return err
		}

		var w io.Writer = os.Stdout
		if *outputFlag == "json" {
			return outputModesJSON(w, modes, detected)
		}

		outputModesText(w, modes, detected)
		return nil
	}

	return cmd
}

func newCacheMountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mount",
		Short: "Restore cache paths from a Namespace volume",
	}

	dryRun := cmd.Flags().Bool("dry_run", !isCI(), "If true, mounting of paths is skipped.")
	cacheRoot := cmd.Flags().String("cache_root", os.Getenv(defaultCacheRootEnv), "Override the root path where cache volumes are mounted.")
	detectModes := cmd.Flags().StringSlice("detect", []string{}, "Detects cache mode(s) based on environment. Supply '*' to enable all detectors.")
	manualModes := cmd.Flags().StringSlice("mode", []string{}, "Explicit cache mode(s) to enable.")
	manualPaths := cmd.Flags().StringSlice("path", []string{}, "Explicit cache path(s) to enable.")
	outputFlag := cmd.Flags().StringP("output", "o", "plain", "Output format: plain or json.")
	evalFile := cmd.Flags().String("eval_file", "", "Write a file that can be sourced to export environment variables.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		mounter, err := cache.NewMounter(*cacheRoot)
		if err != nil {
			return err
		}

		// In dry-run mode, we skip mounting and only report what would be done.
		mounter.DestructiveMode = !*dryRun
		if !mounter.DestructiveMode {
			slog.Info("Dry Run mode enabled.")
		}

		result, err := mounter.Mount(cmd.Context(), cache.MountRequest{
			DetectAllModes: len(*detectModes) == 1 && (*detectModes)[0] == "*",
			DetectModes:    *detectModes,
			ManualModes:    *manualModes,
			ManualPaths:    *manualPaths,
		})
		if err != nil {
			return err
		}

		if *evalFile != "" {
			if err := writeEvalFile(*evalFile, result); err != nil {
				return fmt.Errorf("writing eval file: %w", err)
			}
		}

		var w io.Writer = os.Stdout
		if *outputFlag == "json" {
			return outputMountJSON(w, result)
		}

		outputMountText(w, result)
		return nil
	}

	return cmd
}

func outputModesJSON(w io.Writer, modes, detected mode.Modes) error {
	detectedSet := make(map[string]bool, len(detected))
	for _, m := range detected {
		detectedSet[m.Name()] = true
	}

	result := make(map[string]map[string]bool, len(modes))
	for _, m := range modes {
		result[m.Name()] = map[string]bool{
			"detected": detectedSet[m.Name()],
		}
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{"modes": result})
}

func outputModesText(_ io.Writer, modes, detected mode.Modes) {
	detectedSet := make(map[string]bool, len(detected))
	for _, m := range detected {
		detectedSet[m.Name()] = true
	}

	undetectedSet := make(map[string]bool, len(modes)-len(detected))
	for _, name := range modes.Names() {
		if !detectedSet[name] {
			undetectedSet[name] = true
		}
	}

	slog.Info("Detected:")
	if len(detectedSet) == 0 {
		slog.Info("None")
	} else {
		keys := slices.Collect(maps.Keys(detectedSet))
		slices.Sort(keys)
		slog.Info(fmt.Sprintf("- %s", strings.Join(keys, "\n- ")))
	}

	slog.Info("Undetected:")
	if len(undetectedSet) == 0 {
		slog.Info("None")
	} else {
		keys := slices.Collect(maps.Keys(undetectedSet))
		slices.Sort(keys)
		slog.Info(fmt.Sprintf("- %s", strings.Join(keys, "\n- ")))
	}
}

func writeEvalFile(path string, result cache.MountResponse) error {
	if len(result.Output.AddEnvs) == 0 {
		return nil
	}

	var b strings.Builder
	keys := slices.Sorted(maps.Keys(result.Output.AddEnvs))
	for _, k := range keys {
		fmt.Fprintf(&b, "export %s=%q\n", k, result.Output.AddEnvs[k])
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func outputMountJSON(w io.Writer, result cache.MountResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputMountText(_ io.Writer, result cache.MountResponse) {
	if len(result.Input.Modes) > 0 {
		slog.Info(fmt.Sprintf("Used modes: %v", strings.Join(result.Input.Modes, " ")))
	} else {
		slog.Info("No modes used")
	}

	if len(result.Input.Paths) > 0 {
		slog.Info(fmt.Sprintf("Used paths: %v", strings.Join(result.Input.Paths, ", ")))
	} else {
		slog.Info("No paths used")
	}

	if len(result.Output.Mounts) > 0 {
		slog.Info(fmt.Sprintf("%d directorie(s) mounted", len(result.Output.Mounts)))

		var cacheHits int
		for _, mount := range result.Output.Mounts {
			if mount.CacheHit {
				cacheHits++
			}
		}
		slog.Info(fmt.Sprintf("Cache hit rate: %d/%d", cacheHits, len(result.Output.Mounts)))
	}

	slog.Info(fmt.Sprintf("%s of %s used", result.Output.DiskUsage.Used, result.Output.DiskUsage.Total))
}

// isCI returns true if running in a CI environment.
// Currently supports Github Actions and GitLab CI.
func isCI() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true" || os.Getenv("GITLAB_CI") == "true"
}

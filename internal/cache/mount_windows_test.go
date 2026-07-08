//go:build windows

package cache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/namespacelabs/spacectl/internal/cache"
)

// TestMount_CacheLayoutWindows verifies a drive-letter volume becomes a plain
// path component so the cache path nests under the cache root.
func TestMount_CacheLayoutWindows(t *testing.T) {
	cacheRoot := t.TempDir()

	cases := []struct {
		path string
		rel  string
	}{
		{`C:\Users\x\.gradle\caches`, `C\Users\x\.gradle\caches`},
		{`D:\test`, `D\test`},
		{`c:\lower`, `c\lower`},
		// Relative paths have no volume and nest as-is.
		{`.\target`, `target`},
		{`vendor\cache`, `vendor\cache`},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			require.Equal(t, filepath.Join(cacheRoot, tc.rel), mountCachePath(t, cacheRoot, tc.path))
		})
	}
}

// TestMount_RelativeForwardSlashTargetWindows reproduces the customer failure
// where a forward-slash relative mount path (e.g. "./target", as emitted by the
// Rust provider) is handed to cmd's mklink. Without separator normalization cmd
// reads "/target" as an invalid switch. The junction must be created and resolve
// back to the cache source.
func TestMount_RelativeForwardSlashTargetWindows(t *testing.T) {
	from := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(from, "marker"), []byte("ok"), 0o644))

	// Junctions are created relative to the process working directory, matching
	// how the cache action runs from the workspace root.
	t.Chdir(t.TempDir())

	err := cache.DefaultExecutor{}.Mount(t.Context(), from, "./target")
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join("target", "marker"))
	require.NoError(t, err)
	require.Equal(t, "ok", string(data))
}

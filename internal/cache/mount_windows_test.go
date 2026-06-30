//go:build windows

package cache_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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

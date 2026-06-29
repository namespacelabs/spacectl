//go:build !windows

package cache_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestMount_CacheLayoutUnix guards against regressions: on Unix the cache path
// must remain exactly filepath.Join(CacheRoot, path).
func TestMount_CacheLayoutUnix(t *testing.T) {
	cacheRoot := t.TempDir()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	cases := []struct {
		path string
		want string
	}{
		{"/home/x/.gradle/caches", filepath.Join(cacheRoot, "/home/x/.gradle/caches")},
		{"/root/.cache/go-build", filepath.Join(cacheRoot, "/root/.cache/go-build")},
		{"/Users/x/Library/Caches/ms-playwright", filepath.Join(cacheRoot, "/Users/x/Library/Caches/ms-playwright")},
		{"./target", filepath.Join(cacheRoot, "./target")},
		{"vendor/cache", filepath.Join(cacheRoot, "vendor/cache")},
		// A mid-path colon must NOT be treated as a drive letter on Unix.
		{"weird:colon/name", filepath.Join(cacheRoot, "weird:colon/name")},
		// A leading backslash is a normal filename character on Unix.
		{`\leading-backslash`, filepath.Join(cacheRoot, `\leading-backslash`)},
		// ~ expansion is unchanged by the translation.
		{"~/.cache/foo", filepath.Join(cacheRoot, home, ".cache", "foo")},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			require.Equal(t, tc.want, mountCachePath(t, cacheRoot, tc.path))
		})
	}
}

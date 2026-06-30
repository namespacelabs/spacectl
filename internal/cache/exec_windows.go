//go:build windows

package cache

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceEx = kernel32.NewProc("GetDiskFreeSpaceExW")
)

func (e DefaultExecutor) RemoveAll(name string) error {
	return os.RemoveAll(name)
}

func (e DefaultExecutor) DiskUsage(_ context.Context, path string) (DiskUsage, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return DiskUsage{}, fmt.Errorf("converting path %q: %w", path, err)
	}

	var freeAvailable, totalBytes, totalFree uint64
	r, _, callErr := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)
	if r == 0 {
		return DiskUsage{}, fmt.Errorf("GetDiskFreeSpaceEx %q: %w", path, callErr)
	}

	return DiskUsage{
		Total: humanizeBytes(totalBytes),
		Used:  humanizeBytes(totalBytes - totalFree),
	}, nil
}

func humanizeBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}

	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	val := float64(b) / float64(div)
	suffix := []string{"K", "M", "G", "T", "P", "E"}[exp]
	if val < 10 {
		return fmt.Sprintf("%.1f%s", val, suffix)
	}
	return fmt.Sprintf("%.0f%s", val, suffix)
}

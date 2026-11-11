package testutil

import (
	"os"
	"runtime"
	"testing"
)

// AssertPrivate fails unless path exists and, on Unix, is owner-only
// (no group or other bits). Windows has no Unix modes: Go reports 0666 for
// any writable file and privacy comes from the profile ACL, so there it only
// checks that the file exists (Windows CI, 2026-11-11).
func AssertPrivate(t testing.TB, path, what string) {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0 {
		t.Fatalf("%s not private: %v", what, fi.Mode().Perm())
	}
}

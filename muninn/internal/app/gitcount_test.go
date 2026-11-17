package app_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/scopetest"
)

// ce3dmq: Placement derived the aliases, then PlaceForScope derived them
// again through Places: two project lookups per remembered memory, each
// several git processes (~45 ms apiece on the Windows runner). One lookup.
func TestPlacementLooksUpTheProjectOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the git wrapper is a POSIX sh script")
	}
	f := scopetest.New(t)
	home, _ := os.UserHomeDir()
	p := makePlacementRepo(t, filepath.Join(f.Root, "work", "synthetic-repo"))
	b := f.App.Bound(filepath.Join(home, ".jcode"), p.Root)
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	bin, logFile := t.TempDir(), filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> '" + logFile + "'\nexec '" + real + "' \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, name := range []string{"here", "global"} {
		_ = os.Remove(logFile)
		got, err := b.Placement("", name)
		if err != nil || got.Name != name {
			t.Fatalf("%s: %+v %v", name, got, err)
		}
		raw, _ := os.ReadFile(logFile)
		calls := strings.Split(strings.TrimSpace(string(raw)), "\n")
		lookups := 0
		for _, c := range calls {
			if strings.Contains(c, "rev-list --max-parents=0") {
				lookups++
			}
		}
		if lookups != 1 {
			t.Errorf("Placement(%q) looks the project up %d times, want 1 (%d git processes):\n  %s", name, lookups, len(calls), strings.Join(calls, "\n  "))
		}
	}
}

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// h2yn55 (mizaru 2026-11-18): every hug mem command panicked "slice bounds
// out of range [1:0]" in harness.DiscoverJcode on a home with no ~/.jcode.
// hug mem remember, then recall, on such a home: no panic, exit 0.
func TestMemWithoutJcodeHome(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(base, "hug")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = append(os.Environ(), "GOMAXPROCS=2", "GOFLAGS=-p=2")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	// No HUG_HOME and no JCODE_HOME: discovery runs on the home path,
	// which has no .jcode.
	var env []string
	for _, e := range os.Environ() {
		if k, _, _ := strings.Cut(e, "="); k == "HUG_HOME" || k == "JCODE_HOME" || k == "XDG_DATA_HOME" || k == "XDG_CONFIG_HOME" {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "HOME="+home, "USERPROFILE="+home, "HUG_SYNC=off", "HUG_NO_EMBED=1", "HUG_USER=tester", "HUG_MODEL=test-model",
		"GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GOMAXPROCS=2")
	for _, args := range [][]string{{"mem", "remember", "Synthetic fact: the kettle boils at ninety degrees."}, {"mem", "recall", "kettle boils"}} {
		cmd := exec.Command(bin, args...)
		cmd.Dir, cmd.Env = home, env
		out, err := cmd.CombinedOutput()
		if err != nil || strings.Contains(string(out), "panic") {
			t.Fatalf("hug %v on a home with no .jcode: %v\n%s", args, err, out)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".jcode")); err == nil {
		t.Fatalf("hug mem created ~/.jcode on a home without jcode")
	}
}

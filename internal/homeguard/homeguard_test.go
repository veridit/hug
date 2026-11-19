package homeguard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHome makes a "real" home with a jcode config and a JCODE_INSTANCES
// block in .zshrc, and points HOME at it.
func fakeHome(t *testing.T) (home, inst string) {
	home = t.TempDir()
	inst = filepath.Join(home, "Projects", "Brovs")
	for _, p := range []string{filepath.Join(home, ".jcode", "config.toml"), filepath.Join(inst, ".jcode", "config.toml")} {
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("[hooks]\n"), 0o644)
	}
	os.WriteFile(filepath.Join(home, ".zshrc"), []byte("typeset -gA JCODE_INSTANCES=(\n  brovs   ~/Projects/Brovs\n  other   \"$HOME/x\"\n)\nexport X=1\n"), 0o644)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("JCODE_HOME", "")
	return home, inst
}

// Files covers each jcode instance .zshrc registers, with ~ and $HOME.
func TestFilesReadsInstances(t *testing.T) {
	home, inst := fakeHome(t)
	got := strings.Join(Files(home), "\n")
	for _, want := range []string{filepath.Join(inst, ".jcode", "config.toml"), filepath.Join(home, "x", ".jcode", "mcp.json"), filepath.Join(home, ".claude", "settings.json")} {
		if !strings.Contains(got, want) {
			t.Errorf("Files lacks %s:\n%s", want, got)
		}
	}
}

// While tests run, HOME and the harness homes are a temp folder; after,
// they are back. A test that still writes a real config fails the run.
func TestGuardMovesHomeAndCatchesWrites(t *testing.T) {
	home, inst := fakeHome(t)
	code := Guard(func() int {
		for _, k := range []string{"HOME", "JCODE_HOME", "CODEX_HOME", "XDG_CONFIG_HOME"} {
			if v := os.Getenv(k); v == "" || strings.HasPrefix(v, home) {
				t.Errorf("%s = %q during tests, want a temp folder", k, v)
			}
		}
		return 0
	})
	if code != 0 || os.Getenv("HOME") != home {
		t.Fatalf("clean run: exit %d, HOME %q after", code, os.Getenv("HOME"))
	}
	cfg := filepath.Join(inst, ".jcode", "config.toml")
	code = Guard(func() int {
		os.WriteFile(cfg, []byte("[hooks]\nturn_start = \"hug mem hook jcode\"\n"), 0o644)
		return 0
	})
	if code != 1 {
		t.Fatalf("a run that rewrote %s exited %d, want 1", cfg, code)
	}

	// A test binary re-run as a child of a guarded test (muninn/run's
	// concurrent writers) keeps the HOME its parent test gave it.
	t.Setenv(guarded, "/parent-guard")
	Guard(func() int {
		if os.Getenv("HOME") != home {
			t.Errorf("child HOME %q, want the parent test's %q", os.Getenv("HOME"), home)
		}
		return 0
	})
}

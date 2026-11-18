package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/internal/env"
)

// m1c28r check 3: with an old name set, hug says which HUG_ name to use,
// with a fix: line, on stderr; nothing without one. Old names are spelt in
// parts: the code holds none.
func TestWarnStale(t *testing.T) {
	old := "MUN" + "INN_HOME"
	var b strings.Builder
	warnStale(&b, []string{old + "=/tmp/m", "PATH=/bin"})
	got := b.String()
	if !strings.Contains(got, "hug: hug reads only HUG_* variables: "+old+" is now HUG_HOME") || !strings.Contains(got, "\nfix: export HUG_HOME=\"$"+old+"\"; unset "+old) {
		t.Fatalf("warning:\n%s", got)
	}
	b.Reset()
	warnStale(&b, []string{"HUG_HOME=/tmp/m"})
	if b.Len() != 0 {
		t.Fatalf("warned without old names: %q", b.String())
	}
}

// a0p5kt check 1: a misspelt switch stops the command before it does
// anything. A HUG_ prefixed NO_SYNC=1 on hug mem forget X exits 2, names it,
// and the fix line sets HUG_SYNC=off; the store is not even created. On
// main it ran the forget (2026-11-18: 11 forgets of a store copy synced
// to origin/muninn-notes).
func TestUnknownHUGVariableRefused(t *testing.T) {
	// The misspelt name is spelt in parts so the repository holds no
	// unknown HUG_ name (env.TestEveryHUGNameIsKnown).
	typo := "HUG_" + "NO_SYNC"
	store := filepath.Join(t.TempDir(), "mem")
	t.Setenv("HUG_HOME", store)
	t.Setenv("HUG_NO_EMBED", "1")
	t.Setenv("HUG_SYNC", "off")
	t.Setenv(typo, "1")
	stderr := capture(t, &os.Stderr, func() {
		if got := run("hug", []string{"mem", "forget", "abcd"}); got != 2 {
			t.Errorf("exit %d, want 2", got)
		}
	})
	for _, want := range []string{"hug mem forget: not a hug variable: " + typo + " (nearest: HUG_SYNC)", "\nfix: unset " + typo + "; export HUG_SYNC=off\n"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr lacks %q:\n%s", want, stderr)
		}
	}
	if _, err := os.Stat(store); !os.IsNotExist(err) {
		t.Errorf("the refused command touched the store: %v", err)
	}
	// A hook stays harmless: it does nothing to the store, exits 0, and
	// leaves one line in the hook log naming the variable and the fix.
	capture(t, &os.Stderr, func() {
		if got := run("hug", []string{"mem", "hook", "claude"}); got != 0 {
			t.Errorf("hook exit %d, want 0", got)
		}
	})
	log, err := os.ReadFile(filepath.Join(store, "data", "hook.log"))
	if err != nil || !strings.Contains(string(log), "hug mem hook: refused, did nothing: not a hug variable: "+typo) || !strings.Contains(string(log), "fix: unset "+typo+"; export HUG_SYNC=off") {
		t.Errorf("hook log: %v\n%s", err, log)
	}
	if entries, _ := os.ReadDir(filepath.Join(store, "data")); len(entries) != 1 {
		t.Errorf("the refused hook wrote more than its log line: %v", entries)
	}
	// a0p5kt check 3: help still answers and lists every user-set
	// variable, and no test one.
	out := capture(t, &os.Stdout, func() { run("hug", []string{"help"}) })
	for _, v := range env.Known {
		if v.Kind == env.UserSet && !strings.Contains(out, "  "+v.Name+" ") {
			t.Errorf("hug help does not list %s:\n%s", v.Name, out)
		}
	}
	if strings.Contains(out, "HUG_TEST_SETTLE") {
		t.Errorf("hug help lists a test variable:\n%s", out)
	}
}

// The nearest real variable for common slips, and no refusal for known ones.
func TestRefusalNearest(t *testing.T) {
	p := "HUG_"
	for typo, want := range map[string]string{p + "NO_SYNC": "HUG_SYNC", p + "SYNC_OFF": "HUG_SYNC", p + "HOMEDIR": "HUG_HOME",
		p + "MODLE": "HUG_MODEL", p + "NOEMBED": "HUG_NO_EMBED", p + "USERNAME": "HUG_USER"} {
		u := env.Unknowns([]string{typo + "=1", "HUG_HOME=/x", "PATH=/bin"})
		if len(u) != 1 || u[0].Nearest.Name != want {
			t.Errorf("%s: %+v, want nearest %s", typo, u, want)
		}
	}
	var b strings.Builder
	if refuseUnknown(&b, []string{"HUG_HOME=/x", "HUG_SYNC=off", "HOME=/h"}, "hug") || b.Len() != 0 {
		t.Errorf("refused known variables: %q", b.String())
	}
}

// capture returns what f writes to *std (os.Stdout or os.Stderr).
func capture(t *testing.T, std **os.File, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := *std
	*std = w
	done := make(chan string)
	go func() { b, _ := io.ReadAll(r); done <- string(b) }()
	f()
	*std = old
	w.Close()
	return <-done
}

package cli

import (
	"regexp"
	"strings"
	"testing"

	"github.com/veridit/hug/internal/env"
	"github.com/veridit/hug/internal/throttle"
)

// passPriorities is every priority a pass of a logged, in order.
func passPriorities(t *testing.T, a *App) []string {
	t.Helper()
	var out []string
	for _, m := range regexp.MustCompile(`pass [^,]*, ((?:nice|class) [^)]*)\)`).FindAllStringSubmatch(syncLogOf(t, a), -1) {
		out = append(out, m[1])
	}
	return out
}

// b6jgpn: the detached sync pass (hug task sync --background-pass) runs
// niced like hug mem's background commands, and its git children with it.
// The pass is a real child process; each pass logs the priority it ran at.
// HUG_FULL_SPEED=1 keeps the parent's priority.
func TestBackgroundPassRunsNiced(t *testing.T) {
	if env.Get(env.FullSpeed) != "" {
		t.Skip("HUG_FULL_SPEED is set for this test process")
	}
	t.Parallel()
	a, _, _ := clones(t)
	id := execute(t, a, "new", "Niced", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	settled(t, a)
	execute(t, a, "note", id, "a write starts a pass")
	settled(t, a)
	got := passPriorities(t, a)
	if len(got) == 0 {
		t.Fatalf("no pass logged its priority:\n%s", syncLogOf(t, a))
	}
	want := throttle.Lowered()
	for i, p := range got {
		if p != want {
			t.Fatalf("pass %d ran at %q, want %q (this test process: %q); log:\n%s", i+1, p, want, throttle.Priority(), syncLogOf(t, a))
		}
	}
	before := len(got)
	a.Sync.Env = append(a.Sync.Env, env.FullSpeed+"=1")
	execute(t, a, "note", id, "full speed")
	settled(t, a)
	got = passPriorities(t, a)
	if len(got) == before {
		t.Fatalf("no pass after HUG_FULL_SPEED=1:\n%s", syncLogOf(t, a))
	}
	for _, p := range got[before:] {
		if p != throttle.Priority() {
			t.Fatalf("with %s=1 a pass ran at %q, want this process's %q", env.FullSpeed, p, throttle.Priority())
		}
	}
	if strings.Contains(syncLogOf(t, a), "nice ?") {
		t.Fatalf("a pass could not read its priority:\n%s", syncLogOf(t, a))
	}
}

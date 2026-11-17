package cli

import (
	"strings"
	"testing"

	"github.com/veridit/hug/internal/cond"
)

// 16dxsr: waits through the CLI. dep --add writes a condition; show prints
// it with what hug saw; next skips the task until it holds; lint names a
// condition hug cannot check, with a fix: line.
func TestWaitsThroughCLI(t *testing.T) {
	load := 140.0
	old := cond.LoadAverage
	cond.LoadAverage = func() (float64, bool) { return load, true }
	t.Cleanup(func() { cond.LoadAverage = old })
	a := app(t)
	execute(t, a, "init")
	heavy := execute(t, a, "new", "Heavy run", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	light := execute(t, a, "new", "Light run", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	execute(t, a, "dep", heavy, "--add", "when load below 120 wakes kestrel")
	if code, out := runText(a, "show", heavy); code != 0 || !strings.Contains(out, "waits: when load below 120  [waiting: load is 140.0]  wakes kestrel\n") {
		t.Fatalf("show: %d\n%s", code, out)
	}
	if s := jsonString(t, execute(t, a, "show", heavy)); !strings.Contains(s, `"waits":[{"text":"load below 120","line":`) || !strings.Contains(s, `"met":false,"seen":"load is 140.0","wakes":"kestrel"`) {
		t.Fatalf("show json: %s", s)
	}
	if r := execute(t, a, "next"); r["task"].(Reply)["id"] != light {
		t.Fatalf("next under load: %v", r["task"].(Reply)["id"])
	}
	if code, out := runText(a, "take", heavy); code != 3 || !strings.Contains(out, "when load below 120 (load is 140.0)") || !strings.HasSuffix(out, "fix: hug task show "+heavy+"\n") {
		t.Fatalf("take while waiting: %d\n%s", code, out)
	}
	load = 60
	if r := execute(t, a, "next"); r["task"].(Reply)["id"] != heavy {
		t.Fatalf("next after load fell: %v", r["task"].(Reply)["id"])
	}
	if _, out := runText(a, "show", heavy); !strings.Contains(out, "waits: when load below 120  [met: load is 60.0]  wakes kestrel\n") {
		t.Fatalf("show met:\n%s", out)
	}

	// A condition hug cannot read: refused by dep, flagged by lint when
	// written by hand.
	if code, out := runText(a, "dep", light, "--add", "when the build is green"); code == 0 || !strings.Contains(out, "is not a condition hug can check") || !strings.Contains(out, "fix: hug task dep "+light+" --add 'when load below 120'") {
		t.Fatalf("dep unknown: %d\n%s", code, out)
	}
	// A branch this clone has never seen: lint ID says it cannot be checked.
	execute(t, a, "dep", light, "--add", "when branch feat/nowhere merged")
	_, out := runText(a, "lint", light)
	if !strings.Contains(out, `[unknown-condition] unknown condition: "when branch feat/nowhere merged" cannot be checked here (no branch feat/nowhere in this clone), so `+light+" waits") ||
		!strings.Contains(out, "next: hug task dep "+light+" --remove 'when branch feat/nowhere merged'") {
		t.Fatalf("lint:\n%s", out)
	}
	execute(t, a, "dep", light, "--remove", "when branch feat/nowhere merged")
	if _, out := runText(a, "lint"); strings.Contains(out, "unknown-condition") {
		t.Fatalf("lint after remove:\n%s", out)
	}
}

// A branch condition records the branch tip, so it holds after the branch
// is merged and deleted.
func TestBranchWaitRecordsTip(t *testing.T) {
	a := app(t)
	execute(t, a, "init")
	git(t, a.Cwd, "checkout", "-q", "-b", "feat/w")
	git(t, a.Cwd, "commit", "-q", "--allow-empty", "-m", "w")
	git(t, a.Cwd, "checkout", "-q", "main")
	id := execute(t, a, "new", "After feat/w", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	execute(t, a, "dep", id, "--add", "when branch feat/w merged")
	f := taskFile(t, a, id)
	tip := strings.TrimSpace(git(t, a.Cwd, "rev-parse", "--short=12", "feat/w"))
	if !strings.Contains(f, "\n- when branch feat/w merged at "+tip+"\n") {
		t.Fatalf("file:\n%s", f)
	}
	if _, out := runText(a, "show", id); !strings.Contains(out, "[waiting: branch feat/w is not merged into main]") {
		t.Fatalf("show:\n%s", out)
	}
	git(t, a.Cwd, "merge", "-q", "--ff-only", "feat/w")
	git(t, a.Cwd, "branch", "-q", "-d", "feat/w")
	cond.MemoFor = 0
	t.Cleanup(func() { cond.MemoFor = 10e9 })
	if _, out := runText(a, "show", id); !strings.Contains(out, "[met: branch feat/w is merged into main]") {
		t.Fatalf("show after merge:\n%s", out)
	}
}

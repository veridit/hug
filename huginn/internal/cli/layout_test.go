package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// newerBoardFixture is a board one layout ahead of this hug: hug.toml on
// the task branch says layout = task.Layout+1, committed as a newer hug's
// upgrade would.
func newerBoardFixture(t *testing.T) (*App, string, string) {
	t.Helper()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	dir := filepath.Join(a.Cwd, TaskDir)
	if e := os.WriteFile(filepath.Join(dir, "hug.toml"), []byte(fmt.Sprintf("layout = %d\n", newerLayout)), 0o644); e != nil {
		t.Fatal(e)
	}
	git(t, dir, "add", "hug.toml")
	git(t, dir, "commit", "-q", "-m", fmt.Sprintf("hug upgrade: board layout %d -> %d", task.Layout, newerLayout))
	return a, id, dir
}

// dx9vw2 check 1: a hug older than its board refuses every write before
// touching it, naming both layouts, with the upgrade as its fix. On
// 2026-11-17 a hug from before state: took a state: board's task as
// having no state and sent its agent to hug task take.
func TestOlderHugRefusesToWriteNewerBoard(t *testing.T) {
	t.Parallel()
	a, id, dir := newerBoardFixture(t)
	head := git(t, dir, "rev-parse", "HEAD")
	file := taskFile(t, a, id)
	for _, args := range [][]string{
		{"note", id, "evidence"},
		{"take", id, "--model", "m"},
		{"wait", id, "CI run 9"},
		{"new", "Other", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a"},
		{"set", id, "--priority", "high"},
		{"sync"},
	} {
		code, out := runText(a, args...)
		want := fmt.Sprintf("this board is newer than this hug: the board has layout %d (hug.toml on branch hug/tasks), this hug knows layout %d", newerLayout, task.Layout)
		if code != 1 || !strings.Contains(out, want) || !strings.HasSuffix(out, "fix: brew upgrade veridit/tap/hug\n") {
			t.Fatalf("%v: exit %d, want 1 with %q and the brew fix:\n%s", args, code, want, out)
		}
	}
	if got := git(t, dir, "rev-parse", "HEAD"); got != head {
		t.Fatalf("the board moved: %s -> %s", head, got)
	}
	if got := taskFile(t, a, id); got != file {
		t.Fatalf("the task file changed:\n%s", got)
	}
	if st := git(t, dir, "status", "--porcelain"); st != "" {
		t.Fatalf("the board worktree is dirty:\n%s", st)
	}
}

// dx9vw2 check 2: a read on a newer board still prints, and says once on
// stderr that the board is newer, with the fix; --json carries it in the
// reply instead.
func TestOlderHugWarnsOnceOnRead(t *testing.T) {
	t.Parallel()
	a, id, _ := newerBoardFixture(t)
	for _, args := range [][]string{{"ls"}, {"show", id}, {"next"}} {
		out, errs := &strings.Builder{}, &strings.Builder{}
		a.Out, a.Err = out, errs
		if code := a.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v: exit %d\n%s%s", args, code, out, errs)
		}
		if !strings.Contains(out.String(), id) {
			t.Fatalf("%v printed no task:\n%s", args, out)
		}
		if n := strings.Count(errs.String(), "this board is newer than this hug"); n != 1 || !strings.Contains(errs.String(), "fix: brew upgrade veridit/tap/hug\n") {
			t.Fatalf("%v: stderr says it %d times, want once with the fix:\n%s", args, n, errs)
		}
		if strings.Contains(out.String(), "newer than this hug") {
			t.Fatalf("%v: the warning is on stdout:\n%s", args, out)
		}
	}
	r := execute(t, a, "ls", "--json")
	if w, ok := r["board_newer"].(Reply); !ok || w["board"] != newerLayout || w["fix"] != "brew upgrade veridit/tap/hug" {
		t.Fatalf("--json has no board_newer: %#v", r["board_newer"])
	}
}

// dx9vw2 (crab 20:35): the state: rename is the first layout bump. A board
// already on state: without the key (hug's and Seid's on 2026-11-17) gets
// this hug's layout from hug upgrade --apply, its task files moved to
// C/ID.md (w40cmk) in the same commit, after which a layout-1 hug writes
// nothing; hug task init writes this hug's layout on a new board.
func TestUpgradeMarksStateBoardLayoutTwo(t *testing.T) {
	t.Parallel()
	a := app(t)
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	dir := filepath.Join(a.Cwd, TaskDir)
	if _, e := os.Stat(filepath.Join(dir, "hug.toml")); !os.IsNotExist(e) {
		t.Fatalf("a board made by new has a hug.toml: %v", e)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, "\nstate: ") {
		t.Fatalf("not a state: board:\n%s", f)
	}
	if code, out := runText(a, "upgrade"); code != UpgradePending || !strings.Contains(out, fmt.Sprintf("tasks: would: board layout 1 -> %d in hug.toml", task.Layout)) {
		t.Fatalf("upgrade preview: exit %d\n%s", code, out)
	}
	if code, out := runText(a, "upgrade", "--apply"); code != 0 || !strings.Contains(out, fmt.Sprintf("tasks: board layout 1 -> %d in hug.toml", task.Layout)) {
		t.Fatalf("upgrade --apply: exit %d\n%s", code, out)
	}
	if toml, _ := os.ReadFile(filepath.Join(dir, "hug.toml")); string(toml) != layoutLine {
		t.Fatalf("hug.toml:\n%s", toml)
	}
	if log := git(t, dir, "log", "-1", "--format=%s"); log != upgradedFrom(1) {
		t.Fatalf("commit %q", log)
	}
	older := *a
	older.Layout = 1
	if code, out := runText(&older, "note", id, "x"); code != 1 || !strings.Contains(out, fmt.Sprintf("the board has layout %d (hug.toml on branch hug/tasks), this hug knows layout 1", task.Layout)) {
		t.Fatalf("a layout-1 hug wrote: exit %d\n%s", code, out)
	}
	execute(t, a, "note", id, "this hug writes")

	b := app(t)
	execute(t, b, "init")
	if toml, _ := os.ReadFile(filepath.Join(b.Cwd, TaskDir, "hug.toml")); string(toml) != layoutLine {
		t.Fatalf("init's hug.toml:\n%s", toml)
	}
	if code, out := runText(b, "upgrade"); code != 0 || strings.Contains(out, "board layout") {
		t.Fatalf("upgrade after init: exit %d\n%s", code, out)
	}
}

// dx9vw2: hug upgrade by a newer hug records the layout, after which an
// older hug refuses to write; init's layout-2 board takes this hug's
// writes as before.
func TestUpgradeRecordsBoardLayout(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no")
	execute(t, a, "note", id, "a board at this hug's layout takes writes")
	newer := *a
	newer.Layout = newerLayout
	bump := fmt.Sprintf("board layout %d -> %d in hug.toml", task.Layout, newerLayout)
	if code, out := runText(&newer, "upgrade"); code != UpgradePending || !strings.Contains(out, "tasks: would: "+bump) {
		t.Fatalf("upgrade preview: exit %d\n%s", code, out)
	}
	if code, out := runText(&newer, "upgrade", "--apply"); code != 0 || !strings.Contains(out, "tasks: "+bump) {
		t.Fatalf("upgrade --apply: exit %d\n%s", code, out)
	}
	toml, e := os.ReadFile(filepath.Join(a.Cwd, TaskDir, "hug.toml"))
	if e != nil || !strings.HasPrefix(string(toml), fmt.Sprintf("layout = %d\n\n[people.jhf]\n", newerLayout)) {
		t.Fatalf("hug.toml (%v):\n%s", e, toml)
	}
	if code, out := runText(&newer, "upgrade"); code != 0 || !strings.Contains(out, "tasks: nothing to do") {
		t.Fatalf("a second upgrade: exit %d\n%s", code, out)
	}
	execute(t, &newer, "note", id, "the newer hug writes")
	has := fmt.Sprintf("board has layout %d", newerLayout)
	if code, out := runText(a, "note", id, "x"); code != 1 || !strings.Contains(out, has) {
		t.Fatalf("the older hug wrote: exit %d\n%s", code, out)
	}
	// An older hug's upgrade stops too: it would move the board back.
	if code, out := runText(a, "upgrade", "--apply"); code != 1 || !strings.Contains(out, has) {
		t.Fatalf("the older hug's upgrade: exit %d\n%s", code, out)
	}
}

// 5sc5rw: hug upgrade's memory part runs hug mem upgrade: a preview asks for
// --apply and exits pending, --apply passes it on, and a failure stops.
func TestUpgradeMemoryPartRunsMemUpgrade(t *testing.T) {
	t.Parallel()
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	var calls [][]string
	folded := false
	a.MemUpgrade = func(args []string) (int, string) {
		calls = append(calls, args)
		if folded {
			return 0, "memory: nothing to do\n"
		}
		if len(args) == 1 && args[0] == "--apply" {
			folded = true
			return 0, "memory: folded 3 copies of this machine's own memories in project:rc-x\nmemory: backup memories.json.bak-upgrade-1\n"
		}
		return 5, "memory: would fold 3 copies of this machine's own memories in project:rc-x\nfix: hug mem upgrade --apply\n"
	}
	r := execute(t, a, "upgrade")
	did := strings.Join(r["upgrade"].([]string), "; ")
	if !strings.Contains(did, "memory: would fold 3 copies of this machine's own memories in project:rc-x") || strings.Contains(did, "fix:") || r["exit_code"] != UpgradePending {
		t.Fatalf("preview: %s (exit %v)", did, r["exit_code"])
	}
	// The memory store is this machine's: --apply alone leaves it (f5vx4x).
	r = execute(t, a, "upgrade", "--apply")
	if did = strings.Join(r["upgrade"].([]string), "; "); !strings.Contains(did, "memory: would fold 3 copies") || r["exit_code"] != UpgradePending || r["fix"] != "hug upgrade --apply --machine" {
		t.Fatalf("--apply alone: %s (exit %v, fix %v)", did, r["exit_code"], r["fix"])
	}
	r = execute(t, a, "upgrade", "--apply", "--machine")
	if did = strings.Join(r["upgrade"].([]string), "; "); !strings.Contains(did, "memory: folded 3 copies") || !strings.Contains(did, "memory: backup memories.json.bak-upgrade-1") {
		t.Fatalf("apply: %s", did)
	}
	if r = execute(t, a, "upgrade"); r["exit_code"] != nil || !strings.Contains(strings.Join(r["upgrade"].([]string), "; "), "memory: nothing to do") {
		t.Fatalf("after: %v", r)
	}
	if len(calls) != 4 || len(calls[0]) != 0 || len(calls[1]) != 0 || calls[2][0] != "--apply" {
		t.Fatalf("calls %v", calls)
	}
	a.MemUpgrade = func([]string) (int, string) { return 1, "hug mem: store locked\n" }
	if code, out := runText(a, "upgrade"); code != 1 || !strings.Contains(out, "hug mem upgrade failed: hug mem: store locked") || !strings.HasSuffix(out, "fix: hug mem upgrade\n") {
		t.Fatalf("failure: exit %d\n%s", code, out)
	}
}

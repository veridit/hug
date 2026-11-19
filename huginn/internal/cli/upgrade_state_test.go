package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// asr4t2 + 6ephqk step 4: hug upgrade runs every migration, part by part.
// A plain run previews: it says what it would do ("would: ") and the diff, and changes
// nothing; the real run renames status: to state: and commits; a second run
// says nothing to do for every part.
func TestUpgradeMigratesStateKey(t *testing.T) {
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	id := execute(t, a, "new", "Old file", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	m, _ := globID(filepath.Join(a.Cwd, TaskDir), id)
	b, _ := os.ReadFile(m[0])
	old := strings.Replace(string(b), "\nstate: todo\n", "\nstatus: todo\n", 1)
	if old == string(b) {
		t.Fatalf("new file has no state: todo line:\n%s", b)
	}
	if err := os.WriteFile(m[0], []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	board := filepath.Join(a.Cwd, TaskDir)
	git(t, board, "commit", "-qam", "a board from before state:")
	head := git(t, board, "rev-parse", "HEAD")

	if _, out := runText(a, "lint", id); !strings.Contains(out, "[old-layout] old layout: status: is now state:") || !strings.Contains(out, "hug upgrade") {
		t.Fatalf("lint:\n%s", out)
	}

	r := execute(t, a, "upgrade")
	did := strings.Join(r["upgrade"].([]string), "; ")
	if !strings.Contains(did, "tasks: would: 1 tasks moved to state: (was status:)") || !strings.Contains(did, "layout: nothing to do") {
		t.Fatalf("dry run: %s", did)
	}
	if d, _ := r["diff"].(string); !strings.Contains(d, id+": status: todo -> state: todo") {
		t.Fatalf("dry run diff: %q", d)
	}
	if r["next"] != "hug upgrade --apply" || r["exit_code"] != UpgradePending {
		t.Fatalf("preview next %v exit %v", r["next"], r["exit_code"])
	}
	if after, _ := os.ReadFile(m[0]); string(after) != old || git(t, board, "rev-parse", "HEAD") != head {
		t.Fatal("dry run changed the board")
	}

	r = execute(t, a, "upgrade", "--apply")
	if did = strings.Join(r["upgrade"].([]string), "; "); !strings.Contains(did, "tasks: 1 tasks moved to state: (was status:)") || strings.Contains(did, "would:") {
		t.Fatalf("upgrade: %s", did)
	}
	if r["next"] != "hug setup" {
		t.Fatalf("next %v", r["next"])
	}
	if after := taskFile(t, a, id); after != strings.Replace(old, "\nstatus: todo\n", "\nstate: todo\n", 1) {
		t.Fatalf("migrated:\n%s", after)
	}
	if log := git(t, board, "log", "-1", "--format=%s"); log != "hug upgrade: status: -> state: in 1 tasks" {
		t.Fatalf("commit %q", log)
	}
	if st := git(t, board, "status", "--porcelain"); st != "" {
		t.Fatalf("not committed: %s", st)
	}

	r = execute(t, a, "upgrade", "--apply")
	if did = strings.Join(r["upgrade"].([]string), "; "); did != "layout: nothing to do; tasks: nothing to do; variables: nothing to do; names: nothing to do; memory: nothing to do" {
		t.Fatalf("second upgrade: %s", did)
	}
	if _, ok := r["diff"]; ok {
		t.Fatalf("second upgrade has a diff: %v", r["diff"])
	}
}

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 6ephqk step 4 (crab 13:33): a time written without a zone is a lint
// finding with fix: hug upgrade; hug upgrade writes it in UTC with Z and
// commits; new writes are UTC already.
func TestUpgradeWritesTimesInUTC(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Times", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id)
	m, _ := filepath.Glob(filepath.Join(a.Cwd, TaskDir, "*"+id+"*.md"))
	b, _ := os.ReadFile(m[0])
	if !strings.Contains(string(b), "Z\"\n") {
		t.Fatalf("take wrote no UTC time:\n%s", b)
	}
	// An older hug's zoneless time.
	lines := strings.Split(string(b), "\n")
	for i, l := range lines {
		if strings.HasPrefix(l, "doing_since:") {
			lines[i] = `doing_since: "2026-11-17T14:30"`
		}
	}
	if err := os.WriteFile(m[0], []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(a.Cwd, TaskDir), "commit", "-qam", "old time")
	if _, out := runText(a, "lint"); !strings.Contains(out, `[zoneless-time] time without zone: doing_since is "2026-11-17T14:30"`) || !strings.Contains(out, "fix: hug upgrade") {
		t.Fatalf("lint:\n%s", out)
	}
	if got := strings.Join(execute(t, a, "upgrade")["upgrade"].([]string), "; "); !strings.Contains(got, "times written in UTC in 1 tasks") {
		t.Fatalf("upgrade: %s", got)
	}
	after, _ := os.ReadFile(m[0])
	if !strings.Contains(string(after), "\ndoing_since: \"2026-11-17T") || !strings.Contains(string(after), ":00Z\"\n") {
		t.Fatalf("not converted:\n%s", after)
	}
	if st := git(t, filepath.Join(a.Cwd, TaskDir), "status", "--porcelain"); st != "" {
		t.Fatalf("not committed: %s", st)
	}
	if _, out := runText(a, "lint"); strings.Contains(out, "zoneless-time") {
		t.Fatalf("lint after upgrade:\n%s", out)
	}
}

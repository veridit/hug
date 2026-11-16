package app_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// x1t1t1: a memory remembered or captured in a linked worktree records the
// branch checked out there; on the main checkout's own branch it records none.
func TestMemoriesRecordWorktreeBranch(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	repo := filepath.Join(f.Root, "repo")
	os.MkdirAll(repo, 0o755)
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "commit", "-q", "--allow-empty", "-m", "root")
	wt := filepath.Join(f.Root, "repo-feat")
	gitIn(t, repo, "worktree", "add", "-q", "-b", "feat/x", wt)

	branchOf := func(sid string) string {
		t.Helper()
		mems, err := ingest.LoadNative(app.NativePath())
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range mems {
			if m.SID == sid {
				return m.Branch
			}
		}
		t.Fatalf("memory %s not in the store", sid)
		return ""
	}

	inWorktree := f.App.Bound("", wt)
	u, _, err := inWorktree.Remember(ctx, f.App, "Quartz worktree note remembered on a feature branch", "fact", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := branchOf(u.SID); got != "feat/x" {
		t.Fatalf("remember in worktree: branch %q, want feat/x", got)
	}

	onMain := f.App.Bound("", repo)
	u, _, err = onMain.Remember(ctx, f.App, "Quartz main checkout note on the main branch", "fact", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := branchOf(u.SID); got != "" {
		t.Fatalf("remember on main: branch %q, want none", got)
	}

	fakeCapture(t, f, line("fact", "high", "Quartz capture from a worktree keeps its branch for later cleanup.", ""))
	r, err := inWorktree.Capture(ctx, f.App, captureOpts("worktree-branch"))
	if err != nil || len(r.Actions) == 0 || r.Actions[0].Action != "add" {
		t.Fatalf("capture: %+v %v", r, err)
	}
	mems, _ := ingest.LoadNative(app.NativePath())
	found := false
	for _, m := range mems {
		if m.Content == "Quartz capture from a worktree keeps its branch for later cleanup." {
			found = true
			if m.Branch != "feat/x" {
				t.Fatalf("captured in worktree: branch %q, want feat/x", m.Branch)
			}
		}
	}
	if !found {
		t.Fatal("captured memory not saved")
	}

	if app.WorkBranch(t.TempDir()) != "" {
		t.Fatal("outside git records no branch")
	}
}

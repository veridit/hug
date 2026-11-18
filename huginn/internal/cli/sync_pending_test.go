package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
)

// jj7jpz: a pass whose board does not open returns before Background.Run,
// which is what removes the pending marker its Nudge wrote. The marker
// stayed and settled waited on it for 30 s (CI 44230245084, ubuntu: "open:
// .../two/.hug/tasks is not a working checkout of hug/tasks", then "pending
// true"). Counted, not timed: each real child is waited for by its own log
// line, then the marker is looked at.
func TestPassThatCannotOpenEndsPending(t *testing.T) {
	t.Parallel()
	a, _, _ := clones(t)
	settled(t, a)
	bg, e := gitbranch.BackgroundOf(a.Cwd, TaskBranch)
	if e != nil {
		t.Fatal(e)
	}
	// Git forgets the board's worktree (its admin folder goes), the folder
	// stays: Open refuses it, as on the CI runner.
	board := filepath.Join(a.Cwd, TaskDir)
	dot, e := os.ReadFile(filepath.Join(board, ".git"))
	if e != nil {
		t.Fatal(e)
	}
	admin := strings.TrimSpace(strings.TrimPrefix(string(dot), "gitdir: "))
	if e = os.RemoveAll(admin); e != nil {
		t.Fatal(e)
	}
	const rounds = 5
	left := 0
	for i := 0; i < rounds; i++ {
		if e = bg.Nudge(a.Cwd, a.Sync.Spawn, a.Sync.Env...); e != nil {
			t.Fatal(e)
		}
		ended := until(60*time.Second, func() bool {
			log, _ := os.ReadFile(bg.Log())
			return strings.Count(string(log), "is not a working checkout of "+TaskBranch) >= i+1
		})
		if !ended {
			log, _ := os.ReadFile(bg.Log())
			t.Fatalf("round %d: the pass never reported the board it could not open; log:\n%s", i, log)
		}
		// The child logs, then returns: give its exit a moment to finish
		// whatever it does after the line, by its own pending, not a clock.
		if !until(5*time.Second, func() bool { return !bg.Pending() }) {
			left++
			_ = os.Remove(filepath.Join(filepath.Dir(bg.Log()), strings.TrimSuffix(filepath.Base(bg.Log()), ".log")+".pending"))
		}
	}
	if left > 0 {
		log, _ := os.ReadFile(bg.Log())
		t.Fatalf("%d of %d passes that could not open the board left their pending marker; log:\n%s", left, rounds, log)
	}
}

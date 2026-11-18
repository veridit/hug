package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// Check9's ordinary sync acceptance is deliberately SAME-task. Once both
// clones have layout3, only sync is called: no upgrade/CatchUp can supply
// the merge. Each independent append must reach both clones exactly once.
func TestOrdinarySyncSameTaskAppendsConvergeAfterUpgrade(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		applyCatchup(t, a)
		execute(t, a, "sync")
		execute(t, b, "sync")
		for _, board := range []string{ba, bb} {
			if !strings.Contains(git(t, board, "show", "HEAD:hug.toml"), "layout = 3") {
				t.Fatal("fixture did not upgrade both clones")
			}
		}
		execute(t, a, "note", ids[0], "independent ordinary A append")
		execute(t, b, "note", ids[0], "independent ordinary B append")
		for _, c := range []*App{a, b, a} {
			code, out := runText(c, "sync")
			if code != 0 {
				t.Fatalf("ordinary sync in %s: exit%d\n%s", c.Cwd, code, out)
			}
		}
		for _, c := range []*App{a, b} {
			text := taskFile(t, c, ids[0])
			for _, note := range []string{"independent ordinary A append", "independent ordinary B append"} {
				if n := strings.Count(text, note); n != 1 {
					t.Fatalf("%s has %q %d times:\n%s", c.Cwd, note, n, text)
				}
			}
		}
		if git(t, ba, "rev-parse", "HEAD") != git(t, bb, "rev-parse", "HEAD") {
			t.Fatal("ordinary A-sync/B-sync/A-sync did not converge")
		}
	})
}

func TestOrdinarySyncUnreadableTaskKeepsLocalCommits(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		applyCatchup(t, a)
		execute(t, a, "sync")
		execute(t, b, "sync")
		execute(t, b, "note", ids[1], "keep the earlier local commit")
		execute(t, b, "set", ids[1], "--priority", "low")
		p := strings.TrimSpace(git(t, bb, "ls-files", "--", ids[0][:1]+"/"+ids[0]+".md"))
		oldWrite(t, bb, "unreadable local task", func() {
			writeCatchupFile(t, bb, p, []byte("not task YAML\n"))
		})
		execute(t, a, "note", ids[0], "keep the remote append")
		execute(t, a, "set", ids[1], "--priority", "high")
		execute(t, a, "sync")
		local, remote := git(t, bb, "rev-parse", "HEAD"), git(t, ba, "rev-parse", "HEAD")
		code, out := runText(b, "sync")
		if code == 0 || !strings.Contains(out, "fix:") || !strings.Contains(out, ids[0]) {
			t.Fatalf("unsafe malformed-task sync: exit%d\n%s", code, out)
		}
		if git(t, bb, "rev-parse", "HEAD") != local || git(t, ba, "rev-parse", "HEAD") != remote || git(t, bb, "status", "--porcelain") != "" {
			t.Fatal("failed ordinary sync changed HEAD or left partial replay")
		}
		if got := git(t, bb, "rev-list", "--count", remote+"..HEAD"); got != "3" {
			t.Fatalf("local commits lost: %s", got)
		}
		if strings.Contains(out, "sync: "+ids[1]+" priority:") {
			t.Fatalf("reported an uninstalled private resolution:\n%s", out)
		}
		if !strings.Contains(taskFile(t, b, ids[1]), "keep the earlier local commit") {
			t.Fatal("earlier local content lost")
		}
	})
}

// The Goal edits touch different lines and could text-merge without a Git
// conflict. They are still competing single values, not a hybrid Goal.
func TestOrdinarySyncScalarFieldsKeepBoardAndReportLocal(t *testing.T) {
	catchupPair(t, func(a, b *App, ba, bb string, ids []string) {
		applyCatchup(t, a)
		execute(t, a, "sync")
		execute(t, b, "sync")
		execute(t, a, "set", ids[0], "--goal", "base one\nbase middle\nbase last")
		execute(t, a, "sync")
		execute(t, b, "sync")
		boardGoal := "board first\nbase middle\nbase last"
		localGoal := "base one\nbase middle\nlocal last"
		execute(t, a, "set", ids[0], "--goal", boardGoal, "--priority", "high")
		execute(t, b, "set", ids[0], "--goal", localGoal, "--priority", "low")
		execute(t, a, "sync")
		code, out := runText(b, "sync")
		if code != 0 {
			t.Fatalf("scalar sync: exit%d\n%s", code, out)
		}
		for _, field := range []string{"priority", "## Goal"} {
			if strings.Count(out, "sync: "+ids[0]+" "+field+":") != 1 {
				t.Fatalf("missing one visible %s report:\n%s", field, out)
			}
		}
		execute(t, a, "sync")
		for _, c := range []*App{a, b} {
			text := taskFile(t, c, ids[0])
			d := task.Parse(ids[0][:1]+"/"+ids[0]+".md", []byte(text))
			if strings.TrimSpace(d.Task.Goal) != boardGoal || d.Task.Priority != "high" {
				t.Fatalf("canonical scalar values:\n%s", text)
			}
			if !strings.Contains(text, "> base one\n> base middle\n> local last") || !strings.Contains(text, "low") {
				t.Fatalf("local scalar values lost:\n%s", text)
			}
			before := text
			execute(t, c, "sync")
			if taskFile(t, c, ids[0]) != before {
				t.Fatal("ordinary scalar resolution is not idempotent")
			}
		}
	})
}

func TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade(t *testing.T) {
	a, b, _ := clones(t)
	var id string
	offline(t, a, func() {
		offline(t, b, func() {
			ids := oldLayoutBoard(t, a)
			id = ids[0]
			execute(t, a, "sync")
			execute(t, b, "sync")
			applyCatchup(t, a)
			execute(t, a, "sync")
			execute(t, b, "sync")
			execute(t, a, "note", id, "independent background A append")
			execute(t, b, "note", id, "independent background B append")
			execute(t, a, "sync")
		})
	})
	// This write launches the real detached child, not App.Run("sync").
	execute(t, b, "note", id, "background B triggers the child")
	settled(t, b)
	execute(t, a, "sync")
	for _, c := range []*App{a, b} {
		text := taskFile(t, c, id)
		for _, note := range []string{"independent background A append", "independent background B append", "background B triggers the child"} {
			if strings.Count(text, note) != 1 {
				t.Fatalf("%s lacks one %q:\n%s\nsync log:\n%s", c.Cwd, note, text, syncLogOf(t, b))
			}
		}
	}
	if git(t, filepath.Join(a.Cwd, TaskDir), "rev-parse", "HEAD") != git(t, filepath.Join(b.Cwd, TaskDir), "rev-parse", "HEAD") {
		t.Fatal("background ordinary sync did not converge")
	}
}

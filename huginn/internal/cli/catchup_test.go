package cli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// w40cmk check 9 (crab 04:44): clone A moves the board to layout 3 and
// pushes; clone B, still on an older hug, made 2 notes and 1 rank move
// on layout 2 and has not pushed. B's sync refuses with fix hug upgrade
// and keeps its commits; B's hug upgrade --apply replays them onto the
// remote, the notes in C/ID.md and the move a rank: line, no rename.
// A third clone with nothing unpushed fast-forwards.

// oldWrite is what a hug from before layout 3 did: edit a task file in
// place or rename it, then commit. hug at this version refuses to write
// a layout-2 board, so the test plays the older hug with git.
func oldWrite(t *testing.T, board, msg string, f func()) {
	t.Helper()
	f()
	git(t, board, "add", "-A")
	git(t, board, "commit", "-qm", msg)
}

func oldNote(t *testing.T, board, rel, note string) {
	t.Helper()
	p := filepath.Join(board, filepath.FromSlash(rel))
	b, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	s := string(b)
	if !strings.Contains(s, "## Notes\n") {
		s = strings.TrimRight(s, "\n") + "\n\n## Notes\n"
	}
	if e := os.WriteFile(p, []byte(strings.TrimRight(s, "\n")+"\n\n"+note+"\n"), 0o644); e != nil {
		t.Fatal(e)
	}
}

// oldPath is the file of id on a layout-2 board.
func oldPath(t *testing.T, board, id string) string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(board, "*", "*."+id+".*.md"))
	more, _ := filepath.Glob(filepath.Join(board, "*."+id+".*.md"))
	m = append(m, more...)
	if len(m) != 1 {
		t.Fatalf("old file of %s: %v", id, m)
	}
	rel, _ := filepath.Rel(board, m[0])
	return filepath.ToSlash(rel)
}

func TestCatchUpAcrossLayoutMove(t *testing.T) {
	t.Parallel()
	a, b, bare := clones(t)
	boardA := filepath.Join(a.Cwd, TaskDir)
	boardB := filepath.Join(b.Cwd, TaskDir)
	var ids []string
	offline(t, a, func() {
		offline(t, b, func() {
			ids = oldLayoutBoard(t, a)
			git(t, boardA, "push", "-q", "origin", "HEAD:"+TaskBranch)
			git(t, boardB, "pull", "-q", "--rebase", "origin", TaskBranch)

			// B, on an older hug: two notes and a rank move, unpushed.
			first := oldPath(t, boardB, ids[0])
			oldWrite(t, boardB, "hug task note "+ids[0], func() { oldNote(t, boardB, first, "2026-11-18T06:01 B's first note") })
			second := oldPath(t, boardB, ids[1])
			oldWrite(t, boardB, "hug task note "+ids[1], func() { oldNote(t, boardB, second, "2026-11-18T06:02 B's second note") })
			moved := strings.Replace(second, "b."+ids[1], "e."+ids[1], 1)
			oldWrite(t, boardB, "hug task move "+ids[1], func() {
				if e := os.Rename(filepath.Join(boardB, second), filepath.Join(boardB, moved)); e != nil {
					t.Fatal(e)
				}
			})

			// A moves the board and pushes; it also notes ids[0], so the
			// catch-up has a note on both sides.
			if code, out := runText(a, "upgrade", "--apply"); code != 0 && code != UpgradePending {
				t.Fatalf("A's upgrade: exit %d\n%s", code, out)
			}
			execute(t, a, "note", ids[0], "A's note after the move")
			execute(t, a, "sync")

			// B's sync refuses: nothing rebased, its 3 commits kept.
			headB := git(t, boardB, "rev-parse", "HEAD")
			_, err := b.Execute(context.Background(), []string{"sync"})
			if err == nil {
				t.Fatalf("B's sync went through across the layout move")
			}
			want := "sync stopped: the remote board is layout 3, this clone's 3 unpushed commits are layout 2; nothing was rebased, local commits are kept\nfix: hug upgrade"
			if err.Error() != want {
				t.Fatalf("B's sync:\n%s\nwant:\n%s", err, want)
			}
			if now := git(t, boardB, "rev-parse", "HEAD"); now != headB {
				t.Fatalf("B's HEAD moved: %s", git(t, boardB, "log", "--oneline", "-5"))
			}
			code, out := runText(b, "sync")
			t.Logf("B's sync: exit %d\n%s", code, out)
			if code == 0 || !strings.HasSuffix(out, "fix: hug upgrade\n") {
				t.Fatalf("B's sync: exit %d\n%s", code, out)
			}

			// The preview names each commit to replay, changes nothing.
			code, out = runText(b, "upgrade")
			t.Logf("B's preview: exit %d\n%s", code, out)
			for _, w := range []string{
				"  catch-up: would: the fetched board is translated to layout 3; 3 commits of this clone are replayed onto it, one by one, each with its own message (nothing pushed until the next sync)\n",
				" hug task note " + ids[0] + "\n    ", " hug task note " + ids[1] + "\n    ", " hug task move " + ids[1] + "\n",
				"  tasks: would: run after the catch-up, on the remote's board (hug upgrade shows them then)\n",
			} {
				if !strings.Contains(out, w) {
					t.Fatalf("B's preview lacks %q: exit %d\n%s", w, code, out)
				}
			}
			if code != UpgradePending || git(t, boardB, "rev-parse", "HEAD") != headB {
				t.Fatalf("B's preview: exit %d, HEAD %s", code, git(t, boardB, "log", "--oneline", "-3"))
			}

			// --apply: the remote's history, then B's 3 commits.
			code, out = runText(b, "upgrade", "--apply")
			t.Logf("B's upgrade --apply: exit %d\n%s", code, out)
			if code != 0 && code != UpgradePending {
				t.Fatalf("B's upgrade: exit %d\n%s", code, out)
			}
			remote := git(t, bare, "rev-parse", TaskBranch)
			if git(t, boardB, "merge-base", "HEAD", remote) != remote {
				t.Fatalf("B is not on the remote:\n%s", git(t, boardB, "log", "--oneline", "-6"))
			}
			if got := git(t, boardB, "log", "--format=%s", remote+"..HEAD"); got != "hug task move "+ids[1]+"\nhug task note "+ids[1]+"\nhug task note "+ids[0] {
				t.Fatalf("replayed:\n%s", got)
			}
			// The move is a rank: line, no rename.
			if st := git(t, boardB, "show", "--name-status", "--format=", "HEAD"); st != "M\t"+task.PathFor(ids[1]) {
				t.Fatalf("the move commit: %q", st)
			}
			f1, f2 := taskFile(t, b, ids[0]), taskFile(t, b, ids[1])
			if !strings.Contains(f1, "B's first note") || !strings.Contains(f1, "A's note after the move") {
				t.Fatalf("%s lost a note:\n%s", ids[0], f1)
			}
			if !strings.Contains(f2, "B's second note") || !strings.Contains(f2, "rank: \"e\"") {
				t.Fatalf("%s:\n%s", ids[1], f2)
			}
			for _, id := range ids {
				if m, _ := filepath.Glob(filepath.Join(boardB, "*."+id+".*.md")); len(m) > 0 {
					t.Fatalf("an old-named file is back: %v", m)
				}
			}
			execute(t, b, "sync")
			if got := git(t, bare, "rev-parse", TaskBranch); got != git(t, boardB, "rev-parse", "HEAD") {
				t.Fatalf("B's sync after the catch-up pushed nothing")
			}
			if s := git(t, boardB, "status", "--porcelain"); s != "" {
				t.Fatalf("B left changes: %s", s)
			}
		})
	})
}

// A value changed on both sides (crab 04:44, rule 2): the board's stays,
// this clone's is kept in a note on the task, and the run prints one line
// for it. Nothing stops.
func TestCatchUpKeepsBothSidesValueInNote(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	boardA := filepath.Join(a.Cwd, TaskDir)
	boardB := filepath.Join(b.Cwd, TaskDir)
	offline(t, a, func() {
		offline(t, b, func() {
			ids := oldLayoutBoard(t, a)
			git(t, boardA, "push", "-q", "origin", "HEAD:"+TaskBranch)
			git(t, boardB, "pull", "-q", "--rebase", "origin", TaskBranch)
			p := oldPath(t, boardB, ids[2])
			oldWrite(t, boardB, "hug task set "+ids[2], func() {
				f := filepath.Join(boardB, filepath.FromSlash(p))
				raw, _ := os.ReadFile(f)
				var out []string
				for _, l := range strings.SplitAfter(string(raw), "\n") {
					if strings.HasPrefix(l, "priority: ") {
						l = "priority: \"low\"\n"
					}
					out = append(out, l)
				}
				if s := strings.Join(out, ""); s == string(raw) {
					t.Fatalf("no priority line:\n%s", raw)
				} else if e := os.WriteFile(f, []byte(s), 0o644); e != nil {
					t.Fatal(e)
				}
			})
			if code, out := runText(a, "upgrade", "--apply"); code != 0 && code != UpgradePending {
				t.Fatalf("A's upgrade: exit %d\n%s", code, out)
			}
			execute(t, a, "set", ids[2], "--priority", "high")
			execute(t, a, "sync")
			git(t, boardB, "fetch", "-q", "origin") // explicit known cross-layout catch-up
			code, out := runText(b, "upgrade", "--apply")
			t.Logf("B's upgrade --apply: exit %d\n%s", code, out)
			want := "\n      " + ids[2] + " priority: kept the board's high; this clone's low is in a note on " + ids[2] + "\n"
			if code != 0 || !strings.Contains(out, want) {
				t.Fatalf("B's upgrade lacks %q: exit %d\n%s", want, code, out)
			}
			f := taskFile(t, b, ids[2])
			who := git(t, boardB, "log", "-1", "--format=%an", "--grep", "hug task set "+ids[2])
			if !strings.Contains(f, "priority: \"high\"") || !strings.Contains(f, "\n"+who+" set priority low here at ") || !strings.Contains(f, "; the board had high; kept high\n") {
				t.Fatalf("%s:\n%s", ids[2], f)
			}
		})
	})
}

// A file hug cannot read, changed on both sides (rule 3): the only stop.
// HEAD and the work tree are back as they were, every local commit kept,
// and the fix names the commit and the file.
func TestCatchUpStopsOnUnreadableFileAndRestores(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	boardA := filepath.Join(a.Cwd, TaskDir)
	boardB := filepath.Join(b.Cwd, TaskDir)
	offline(t, a, func() {
		offline(t, b, func() {
			ids := oldLayoutBoard(t, a)
			git(t, boardA, "push", "-q", "origin", "HEAD:"+TaskBranch)
			git(t, boardB, "pull", "-q", "--rebase", "origin", TaskBranch)
			first := oldPath(t, boardB, ids[0])
			oldWrite(t, boardB, "hug task note "+ids[0], func() {
				oldNote(t, boardB, first, "fine")
				if e := os.WriteFile(filepath.Join(boardB, "new-local-attachment.txt"), []byte("local attachment\n"), 0o644); e != nil {
					t.Fatal(e)
				}
			})
			p := oldPath(t, boardB, ids[2])
			oldWrite(t, boardB, "edit by hand "+ids[2], func() {
				if e := os.WriteFile(filepath.Join(boardB, filepath.FromSlash(p)), []byte("no front matter, typed by hand\n"), 0o644); e != nil {
					t.Fatal(e)
				}
			})
			if code, out := runText(a, "upgrade", "--apply"); code != 0 && code != UpgradePending {
				t.Fatalf("A's upgrade: exit %d\n%s", code, out)
			}
			execute(t, a, "note", ids[2], "A's note")
			execute(t, a, "sync")
			git(t, boardB, "fetch", "-q", "origin") // explicit known cross-layout catch-up
			head := git(t, boardB, "rev-parse", "HEAD")
			bad := git(t, boardB, "rev-parse", "--short=12", "HEAD")
			code, out := runText(b, "upgrade", "--apply")
			t.Logf("B's upgrade --apply: exit %d\n%s", code, out)
			wantStart := "catch-up stopped at " + bad + " \"edit by hand " + ids[2] + "\": " + task.PathFor(ids[2]) + ": hug cannot read this file (no front matter, or front matter that is not YAML); HEAD is back where it was, 2 local commits kept, nothing pushed\n"
			wantEnd := "; HEAD is back where it was, 2 local commits kept, nothing pushed\nfix: git -C "
			q := "'"
			if runtime.GOOS == "windows" {
				q = "\""
			}
			if code == 0 || !strings.Contains(out, wantStart) || !strings.Contains(out, wantEnd) || !strings.HasSuffix(out, " show "+q+head+":"+p+q+"\n") {
				t.Fatalf("exit %d, want %q ... %q:\n%s", code, wantStart, wantEnd, out)
			}
			if now := git(t, boardB, "rev-parse", "HEAD"); now != head {
				t.Fatalf("HEAD moved: %s", git(t, boardB, "log", "--oneline", "-4"))
			}
			if s := git(t, boardB, "status", "--porcelain"); s != "" {
				t.Fatalf("work tree left changed: %s", s)
			}
			if _, e := os.Stat(filepath.Join(boardB, filepath.FromSlash(p))); e != nil {
				t.Fatalf("B's old-named file is gone: %v", e)
			}
		})
	})
}

// Both clones moved the board on their own (each ran hug upgrade): B's
// sync stops all the same, and its catch-up leaves its own move out and
// replays the rest.
func TestCatchUpWhenBothClonesMoved(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	boardA := filepath.Join(a.Cwd, TaskDir)
	boardB := filepath.Join(b.Cwd, TaskDir)
	offline(t, a, func() {
		offline(t, b, func() {
			ids := oldLayoutBoard(t, a)
			git(t, boardA, "push", "-q", "origin", "HEAD:"+TaskBranch)
			git(t, boardB, "pull", "-q", "--rebase", "origin", TaskBranch)
			first := oldPath(t, boardB, ids[0])
			oldWrite(t, boardB, "hug task note "+ids[0], func() { oldNote(t, boardB, first, "2026-11-18T06:01 B before its move") })
			for _, c := range []*App{a, b} {
				if code, out := runText(c, "upgrade", "--apply"); code != 0 && code != UpgradePending {
					t.Fatalf("upgrade: exit %d\n%s", code, out)
				}
			}
			execute(t, b, "note", ids[1], "B after its move")
			execute(t, a, "sync")
			_, err := b.Execute(context.Background(), []string{"sync"})
			want := "sync stopped: this clone and the remote each moved the board from layout 2 to 3 on their own, and this clone's 3 unpushed commits include its own move; nothing was rebased, local commits are kept\nfix: hug upgrade"
			if err == nil || err.Error() != want {
				t.Fatalf("B's sync: %v\nwant:\n%s", err, want)
			}
			code, out := runText(b, "upgrade", "--apply")
			t.Logf("B's upgrade --apply: exit %d\n%s", code, out)
			if code != 0 || !strings.Contains(out, "hug upgrade: 4 tasks to C/ID.md, rank: and group: in the file (layout 3): not replayed (this clone's own layout move; the remote's stands)") {
				t.Fatalf("exit %d\n%s", code, out)
			}
			if f := taskFile(t, b, ids[0]); !strings.Contains(f, "B before its move") {
				t.Fatalf("%s:\n%s", ids[0], f)
			}
			if f := taskFile(t, b, ids[1]); !strings.Contains(f, "B after its move") {
				t.Fatalf("%s:\n%s", ids[1], f)
			}
			execute(t, b, "sync")
		})
	})
}

func TestCatchUpLayoutCommitAlsoKeepsItsNote(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	boardA := filepath.Join(a.Cwd, TaskDir)
	boardB := filepath.Join(b.Cwd, TaskDir)
	offline(t, a, func() {
		offline(t, b, func() {
			ids := oldLayoutBoard(t, a)
			git(t, boardA, "push", "-q", "origin", "HEAD:"+TaskBranch)
			git(t, boardB, "pull", "-q", "--rebase", "origin", TaskBranch)
			for _, c := range []*App{a, b} {
				if code, out := runText(c, "upgrade", "--apply"); code != 0 && code != UpgradePending {
					t.Fatalf("upgrade: exit %d\n%s", code, out)
				}
			}
			oldNote(t, boardB, task.PathFor(ids[0]), "B's note in its layout-changing commit")
			git(t, boardB, "add", "-A")
			git(t, boardB, "commit", "--amend", "--no-edit", "-q")
			execute(t, a, "sync")
			if code, out := runText(b, "sync"); code == 0 || !strings.Contains(out, "fix: hug upgrade") {
				t.Fatalf("mixed layout sync refusal: exit %d\n%s", code, out)
			}
			code, out := runText(b, "upgrade")
			if code != UpgradePending || strings.Contains(out, "not replayed") {
				t.Fatalf("mixed layout commit preview: exit %d\n%s", code, out)
			}
			code, out = runText(b, "upgrade", "--apply")
			if code != 0 || strings.Contains(out, "not replayed") {
				t.Fatalf("mixed layout commit: exit %d\n%s", code, out)
			}
			if f := taskFile(t, b, ids[0]); strings.Count(f, "B's note in its layout-changing commit") != 1 {
				t.Fatalf("mixed layout commit lost or repeated its note:\n%s", f)
			}
			if got := git(t, boardB, "log", "--format=%s", "origin/"+TaskBranch+"..HEAD"); got != "hug upgrade: 4 tasks to C/ID.md, rank: and group: in the file (layout 3)" {
				t.Fatalf("mixed layout commit message lost: %s", got)
			}
		})
	})
}

// A clone with nothing unpushed is fast-forwarded across the move.
func TestReadOnlyCloneFastForwardsAcrossLayoutMove(t *testing.T) {
	t.Parallel()
	a, b, _ := clones(t)
	boardA := filepath.Join(a.Cwd, TaskDir)
	boardB := filepath.Join(b.Cwd, TaskDir)
	offline(t, a, func() {
		offline(t, b, func() {
			ids := oldLayoutBoard(t, a)
			git(t, boardA, "push", "-q", "origin", "HEAD:"+TaskBranch)
			git(t, boardB, "pull", "-q", "--rebase", "origin", TaskBranch)
			if code, out := runText(a, "upgrade", "--apply"); code != 0 && code != UpgradePending {
				t.Fatalf("A's upgrade: exit %d\n%s", code, out)
			}
			execute(t, a, "sync")
			execute(t, b, "sync")
			if git(t, boardB, "rev-parse", "HEAD") != git(t, boardA, "rev-parse", "HEAD") {
				t.Fatalf("B did not fast-forward")
			}
			if f := taskFile(t, b, ids[0]); !strings.Contains(f, "rank: \"a\"") {
				t.Fatalf("B's file:\n%s", f)
			}
			// Both on layout 3 now: writes from both sides sync as before
			// (check 9's second half), each clone ending with both.
			execute(t, a, "note", ids[0], "A on layout 3")
			execute(t, b, "note", ids[1], "B on layout 3")
			execute(t, b, "move", ids[2], "--top")
			execute(t, a, "sync")
			execute(t, b, "sync")
			execute(t, a, "sync")
			for _, c := range []*App{a, b} {
				if !strings.Contains(taskFile(t, c, ids[0]), "A on layout 3") || !strings.Contains(taskFile(t, c, ids[1]), "B on layout 3") {
					t.Fatalf("a write is missing in %s", c.Cwd)
				}
			}
			if git(t, boardA, "rev-parse", "HEAD") != git(t, boardB, "rev-parse", "HEAD") {
				t.Fatalf("clones differ after sync")
			}
			if n := git(t, boardB, "log", "--diff-filter=R", "--format=%h", "-1", "--", "."); n != "" && git(t, boardB, "log", "-1", "--format=%s", n) != "hug upgrade: 4 tasks to C/ID.md, rank: and group: in the file (layout 3)" {
				t.Fatalf("a rename after the move: %s", git(t, boardB, "log", "-1", "--format=%s", n))
			}
		})
	})
}

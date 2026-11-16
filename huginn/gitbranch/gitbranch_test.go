package gitbranch

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func command(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid", "GIT_TERMINAL_PROMPT=0")
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %s: %v", args, b, e)
	}
	return strings.TrimSpace(string(b))
}
func must(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func writeFile(t *testing.T, r *Repository, path, body string) {
	t.Helper()
	must(t, os.MkdirAll(filepath.Dir(filepath.Join(r.Dir, path)), 0755))
	must(t, os.WriteFile(filepath.Join(r.Dir, path), []byte(body), 0644))
}
func openRepo(t *testing.T, root, branch, dir string) *Repository {
	t.Helper()
	r, e := Open(root, branch, dir, WithIdentity("Test", "test@example.invalid"))
	must(t, e)
	return r
}
func clean(t *testing.T, dir string) {
	t.Helper()
	if s := command(t, dir, "status", "--porcelain"); s != "" {
		t.Fatalf("dirty %s: %s", dir, s)
	}
}
func fixture(t *testing.T, remote bool) (string, string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	base := t.TempDir()
	root := filepath.Join(base, "one")
	must(t, os.MkdirAll(root, 0755))
	command(t, root, "init", "-b", "main")
	command(t, root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "--allow-empty", "-m", "main")
	bare := ""
	if remote {
		bare = filepath.Join(base, "remote.git")
		command(t, base, "init", "--bare", bare)
		command(t, root, "remote", "add", "origin", bare)
		command(t, root, "push", "origin", "main")
		command(t, bare, "symbolic-ref", "HEAD", "refs/heads/main")
	}
	return root, bare
}
func pair(t *testing.T) (*Repository, *Repository) {
	t.Helper()
	root, bare := fixture(t, true)
	a := openRepo(t, root, "huginn", ".huginn")
	writeFile(t, a, "task.md", "unclaimed\n")
	must(t, a.Sync())
	two := filepath.Join(filepath.Dir(root), "two")
	command(t, filepath.Dir(root), "clone", bare, two)
	b := openRepo(t, two, "huginn", ".huginn")
	clean(t, root)
	clean(t, two)
	return a, b
}
func TestInitIdempotentSubdirectoryAndMuninn(t *testing.T) {
	root, _ := fixture(t, true)
	sub := filepath.Join(root, "deep", "sub")
	must(t, os.MkdirAll(sub, 0755))
	t.Chdir(sub)
	a := openRepo(t, ".", "huginn", ".huginn")
	h, e := a.Head()
	must(t, e)
	if h != "" {
		t.Fatal("expected unborn branch")
	}
	writeFile(t, a, "task.md", "task\n")
	must(t, a.Sync())
	h, e = a.Head()
	must(t, e)
	exclude := filepath.Join(root, ".git", "info", "exclude")
	before, e := os.ReadFile(exclude)
	must(t, e)
	again := openRepo(t, sub, "huginn", ".huginn")
	after, e := os.ReadFile(exclude)
	must(t, e)
	h2, e := again.Head()
	must(t, e)
	if string(before) != string(after) || h != h2 {
		t.Fatal("init not idempotent")
	}
	m := openRepo(t, sub, "muninn", ".muninn")
	writeFile(t, m, "memory.md", "memory\n")
	must(t, m.Sync())
	if _, e = os.Stat(filepath.Join(a.Dir, "memory.md")); !os.IsNotExist(e) {
		t.Fatal("branches interfere")
	}
	clean(t, root)
	clean(t, a.Dir)
	clean(t, m.Dir)
	linked := filepath.Join(filepath.Dir(root), "linked")
	command(t, root, "worktree", "add", "-b", "other", linked, "main")
	r := openRepo(t, linked, "linked-notes", ".notes")
	writeFile(t, r, "note.md", "linked\n")
	must(t, r.Sync())
	clean(t, linked)
	clean(t, root)
}
func TestDifferentFileSync(t *testing.T) {
	a, b := pair(t)
	writeFile(t, a, "alice.md", "alice\n")
	writeFile(t, b, "bob.md", "bob\n")
	must(t, a.Sync())
	must(t, b.Sync())
	must(t, a.Sync())
	for _, r := range []*Repository{a, b} {
		for _, p := range []string{"task.md", "alice.md", "bob.md"} {
			if _, e := os.Stat(filepath.Join(r.Dir, p)); e != nil {
				t.Fatal(e)
			}
		}
		clean(t, r.Dir)
		clean(t, r.Root)
	}
}
func TestTakeRace(t *testing.T) {
	a, b := pair(t)
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i, r := range []*Repository{a, b} {
		name := []string{"alice", "bob"}[i]
		wg.Add(1)
		go func(r *Repository, name string) {
			defer wg.Done()
			errs <- r.Update(func(r *Repository) error {
				body, e := os.ReadFile(filepath.Join(r.Dir, "task.md"))
				if e != nil {
					return e
				}
				if string(body) != "unclaimed\n" {
					return errors.New("already taken")
				}
				ready <- struct{}{}
				<-release
				return os.WriteFile(filepath.Join(r.Dir, "task.md"), []byte(name+"\n"), 0644)
			})
		}(r, name)
	}
	for range 2 {
		select {
		case <-ready:
		case e := <-errs:
			t.Fatalf("update failed before race barrier: %v", e)
		case <-time.After(30 * time.Second):
			t.Fatal("update did not reach race barrier")
		}
	}
	unblock()
	wg.Wait()
	close(errs)
	wins, races := 0, 0
	for e := range errs {
		var raced *ErrRaced
		if e == nil {
			wins++
		} else if errors.As(e, &raced) {
			races++
			if raced.RemoteHead == "" {
				t.Fatal("missing remote head")
			}
		} else {
			t.Fatalf("unexpected: %v", e)
		}
	}
	if wins != 1 || races != 1 {
		t.Fatalf("wins=%d races=%d", wins, races)
	}
	ab, e := os.ReadFile(filepath.Join(a.Dir, "task.md"))
	must(t, e)
	bb, e := os.ReadFile(filepath.Join(b.Dir, "task.md"))
	must(t, e)
	if string(ab) != string(bb) {
		t.Fatalf("loser did not restore winner: %s %s", ab, bb)
	}
	for _, r := range []*Repository{a, b} {
		clean(t, r.Dir)
		clean(t, r.Root)
	}
}
func TestConflictAbortsRebase(t *testing.T) {
	a, b := pair(t)
	writeFile(t, a, "task.md", "alice\n")
	writeFile(t, b, "task.md", "bob\n")
	must(t, a.Sync())
	e := b.Sync()
	var conflict *ConflictError
	if !errors.As(e, &conflict) || len(conflict.Files) != 1 || conflict.Files[0] != "task.md" {
		t.Fatalf("expected typed conflict: %#v", e)
	}
	clean(t, b.Dir)
	gitdir := command(t, b.Dir, "rev-parse", "--absolute-git-dir")
	for _, p := range []string{"rebase-merge", "rebase-apply"} {
		if _, e := os.Stat(filepath.Join(gitdir, p)); !os.IsNotExist(e) {
			t.Fatalf("half rebase: %s", p)
		}
	}
	body, e := os.ReadFile(filepath.Join(b.Dir, "task.md"))
	must(t, e)
	if string(body) != "bob\n" {
		t.Fatal("local commit lost")
	}
	clean(t, a.Root)
	clean(t, b.Root)
}
func TestCommitPathsAndRename(t *testing.T) {
	root, _ := fixture(t, false)
	r := openRepo(t, root, "notes", ".notes")
	writeFile(t, r, "a.md", "some unique contents\n")
	writeFile(t, r, "b.md", "other contents\n")
	must(t, r.Commit([]string{"a.md"}, "one"))
	h, e := r.Head()
	must(t, e)
	initial, e := r.DiffNameStatus("", h)
	must(t, e)
	if len(initial) != 1 || initial[0].Path != "a.md" {
		t.Fatalf("initial: %#v", initial)
	}
	must(t, r.Commit(nil, "two"))
	old, e := r.Head()
	must(t, e)
	must(t, os.Rename(filepath.Join(r.Dir, "a.md"), filepath.Join(r.Dir, " new name.md")))
	must(t, r.Commit(nil, "rename"))
	now, e := r.Head()
	must(t, e)
	diff, e := r.DiffNameStatus(old, now)
	must(t, e)
	if len(diff) != 1 || diff[0].Status != "R100" || diff[0].OldPath != "a.md" || diff[0].Path != " new name.md" {
		t.Fatalf("rename: %#v", diff)
	}
	must(t, r.Update(func(r *Repository) error {
		return os.WriteFile(filepath.Join(r.Dir, "local.md"), []byte("local"), 0644)
	}))
	clean(t, r.Dir)
	clean(t, root)
}

// The host repo's hooks must not run on the data branch (seid-lang,
// 2026-11-09: a shared core.hooksPath pre-commit refused huginn's import).
func TestHostHooksDoNotBlockDataBranch(t *testing.T) {
	root, _ := fixture(t, false)
	hooks := filepath.Join(root, ".githooks")
	must(t, os.MkdirAll(hooks, 0o755))
	must(t, os.WriteFile(filepath.Join(hooks, "pre-commit"), []byte("#!/bin/sh\necho BLOCKED >&2\nexit 1\n"), 0o755))
	command(t, root, "config", "core.hooksPath", hooks)
	r := openRepo(t, root, "notes", ".notes")
	writeFile(t, r, "a.md", "data\n")
	must(t, r.Commit(nil, "data"))
	if _, e := r.Head(); e != nil {
		t.Fatal(e)
	}
}

// 065wwn: a linked worktree (git worktree add --detach) uses the branch's one
// checkout in the main worktree, whichever side creates it first, and never
// adds a second worktree for it.
func TestLinkedWorktreeSharesMainCheckout(t *testing.T) {
	for _, first := range []string{"main", "linked"} {
		t.Run(first+" first", func(t *testing.T) {
			root, _ := fixture(t, false)
			linked := filepath.Join(filepath.Dir(root), "linked")
			command(t, root, "worktree", "add", "--detach", linked, "HEAD")
			want := filepath.Join(root, ".hug", "tasks")
			if phys, e := filepath.EvalSymlinks(root); e == nil {
				want = filepath.Join(phys, ".hug", "tasks")
			}
			order := []string{root, linked}
			if first == "linked" {
				order = []string{linked, root}
			}
			var dirs []string
			for _, at := range append(order, filepath.Join(linked)) {
				r := openRepo(t, at, "hug/tasks", ".hug/tasks")
				dirs = append(dirs, r.Dir)
				writeFile(t, r, filepath.Base(at)+".md", "from "+at+"\n")
				must(t, r.Commit(nil, "from "+at))
			}
			for _, d := range dirs {
				if d != want {
					t.Fatalf("Dir %s, want the main worktree's %s", d, want)
				}
			}
			if _, e := os.Stat(filepath.Join(linked, ".hug")); !os.IsNotExist(e) {
				t.Fatalf("linked worktree got its own .hug: %v", e)
			}
			list := command(t, root, "worktree", "list", "--porcelain")
			if n := strings.Count(list, "worktree "); n != 3 {
				t.Fatalf("want 3 worktrees (main, linked, one task checkout), got %d:\n%s", n, list)
			}
			if n := strings.Count(list, "branch refs/heads/hug/tasks"); n != 1 {
				t.Fatalf("hug/tasks checked out %d times:\n%s", n, list)
			}
			for _, f := range []string{"one.md", "linked.md"} {
				if _, e := os.Stat(filepath.Join(want, f)); e != nil {
					t.Fatal(e)
				}
			}
			main, e := MainRoot(filepath.Join(linked))
			must(t, e)
			if filepath.Join(main, ".hug", "tasks") != want {
				t.Fatalf("MainRoot(linked) = %s", main)
			}
		})
	}
}

// A checkout whose folder was deleted by hand is reported with a fix that
// works: git worktree prune, after which Open checks it out again.
func TestMissingCheckoutFixIsPrune(t *testing.T) {
	root, _ := fixture(t, false)
	r := openRepo(t, root, "hug/tasks", ".hug/tasks")
	writeFile(t, r, "a.md", "a\n")
	must(t, r.Commit(nil, "a"))
	linked := filepath.Join(filepath.Dir(root), "linked")
	command(t, root, "worktree", "add", "--detach", linked, "HEAD")
	must(t, os.RemoveAll(r.Dir))
	_, err := Open(linked, "hug/tasks", ".hug/tasks")
	var typed *Error
	if !errors.As(err, &typed) || !strings.HasSuffix(typed.Fix, " worktree prune") {
		t.Fatalf("want prune fix, got %v", err)
	}
	c := exec.Command("sh", "-c", typed.Fix)
	if out, e := c.CombinedOutput(); e != nil {
		t.Fatalf("fix %q: %s %v", typed.Fix, out, e)
	}
	again := openRepo(t, linked, "hug/tasks", ".hug/tasks")
	if _, e := os.Stat(filepath.Join(again.Dir, "a.md")); e != nil {
		t.Fatal(e)
	}
	// A folder git no longer knows as a worktree (pruned while moved away):
	// the fix moves it aside and checks the branch out again.
	aside := filepath.Join(filepath.Dir(root), "aside")
	must(t, os.Rename(again.Dir, aside))
	command(t, root, "worktree", "prune")
	must(t, os.Rename(aside, again.Dir))
	_, err = Open(linked, "hug/tasks", ".hug/tasks")
	if !errors.As(err, &typed) || !strings.Contains(typed.Fix, " worktree add ") {
		t.Fatalf("want move-aside fix, got %v", err)
	}
	if out, e := exec.Command("sh", "-c", typed.Fix).CombinedOutput(); e != nil {
		t.Fatalf("fix %q: %s %v", typed.Fix, out, e)
	}
	third := openRepo(t, linked, "hug/tasks", ".hug/tasks")
	if _, e := os.Stat(filepath.Join(third.Dir, "a.md")); e != nil {
		t.Fatal(e)
	}
}

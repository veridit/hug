package gitbranch

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fcf4fg: Seid, 2026-11-16, a sync interrupted mid-rebase left rebase-merge
// holding only an autostash, and every hug write failed until someone ran
// git rebase --quit and committed by hand.

// stuckRebase makes b's board hold an interrupted hug rebase: b has a
// local commit, an uncommitted edit, and a remote commit to rebase onto.
// rebase.autoStash (user config, as in Seid) stashes the edit; a smudge
// filter that fails stops git while it resets the files. shape "autostash"
// leaves rebase-merge with only the autostash (git died resetting for the
// autostash, Seid's case); "pick" leaves a full rebase-merge (git died
// writing the replayed commit).
func stuckRebase(t *testing.T, shape string) (a, b *Repository, gitdir string) {
	t.Helper()
	a, b = pair(t)
	writeFile(t, a, "remote.md", "from a\n")
	must(t, a.Sync())
	writeFile(t, b, "local.md", "committed in b\n")
	must(t, b.Commit(nil, "b local"))
	_, err := b.fetch()
	must(t, err)
	writeFile(t, b, "task.md", "edited in b, not committed\n")
	gitdir = command(t, b.Dir, "rev-parse", "--absolute-git-dir")
	command(t, b.Dir, "config", "rebase.autoStash", "true")
	command(t, b.Dir, "config", "filter.die.clean", "cat")
	command(t, b.Dir, "config", "filter.die.smudge", "exit 1")
	command(t, b.Dir, "config", "filter.die.required", "true")
	attr := "task.md filter=die\n"
	if shape == "pick" {
		// The pick writes local.md: git stops there with its state, and
		// hug's own abort fails on the same file, as a killed sync's never runs.
		attr = "local.md filter=die\n"
	}
	info := filepath.Join(command(t, b.Dir, "rev-parse", "--git-common-dir"), "info")
	if !filepath.IsAbs(info) {
		info = filepath.Join(b.Dir, info)
	}
	must(t, os.MkdirAll(info, 0o755))
	must(t, os.WriteFile(filepath.Join(info, "attributes"), []byte(attr), 0o644))
	BeforeRebase = func(string) {}
	defer func() { BeforeRebase = nil }()
	if err = b.integrate(); err == nil {
		t.Fatal("the rebase with a failing filter succeeded")
	}
	// The filter is gone, as once the killed process is.
	must(t, os.Remove(filepath.Join(info, "attributes")))
	entries, _ := os.ReadDir(filepath.Join(gitdir, "rebase-merge"))
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if shape == "autostash" && strings.Join(names, " ") != "autostash" {
		t.Fatalf("rebase-merge holds %v, want only autostash", names)
	}
	if shape == "pick" && !strings.Contains(" "+strings.Join(names, " ")+" ", " head-name ") {
		t.Fatalf("rebase-merge holds %v, want a full rebase state", names)
	}
	return a, b, gitdir
}

func TestInterruptedHugRebaseIsEnded(t *testing.T) {
	for _, shape := range []string{"autostash", "pick"} {
		t.Run(shape, func(t *testing.T) {
			a, b, gitdir := stuckRebase(t, shape)
			var lines []string
			b.OnRepaired = func(l string) { lines = append(lines, l) }
			// What every write found on 2026-11-16, before the repair.
			var busy *Error
			if err := b.checkIdle(); !errors.As(err, &busy) || busy.Op != "worktree busy" {
				t.Fatalf("stuck state is not busy: %v", err)
			}
			must(t, b.Recover())
			if _, err := os.Stat(filepath.Join(gitdir, "rebase-merge")); !os.IsNotExist(err) {
				t.Fatalf("rebase-merge left after recovery: %v", err)
			}
			if head := command(t, b.Dir, "symbolic-ref", "HEAD"); head != "refs/heads/huginn" {
				t.Fatalf("HEAD %s after recovery", head)
			}
			if got, _ := os.ReadFile(filepath.Join(b.Dir, "task.md")); string(got) != "edited in b, not committed\n" {
				t.Fatalf("autostashed edit not back: %q", got)
			}
			if len(lines) != 1 || !strings.Contains(lines[0], "ended an interrupted hug rebase") || !strings.Contains(lines[0], "stashed edits re-applied") {
				t.Fatalf("repair lines: %q", lines)
			}
			// The write goes on: the edit and the local commit reach the remote.
			must(t, b.Sync())
			must(t, a.Sync())
			for _, f := range []string{"local.md", "task.md", "remote.md"} {
				if _, err := os.Stat(filepath.Join(a.Dir, f)); err != nil {
					t.Fatalf("%s missing on the other clone: %v", f, err)
				}
			}
			if got, _ := os.ReadFile(filepath.Join(a.Dir, "task.md")); string(got) != "edited in b, not committed\n" {
				t.Fatalf("other clone has task.md %q", got)
			}
			if s := command(t, b.Dir, "stash", "list"); s != "" {
				t.Fatalf("stash left: %s", s)
			}
		})
	}
}

// A rebase hug did not start (a person's, another tool's) still stops the
// write with the fix line, and is not touched.
func TestForeignRebaseStillRefuses(t *testing.T) {
	cases := map[string]func(t *testing.T, b *Repository, gitdir string){
		"no marker": func(t *testing.T, b *Repository, gitdir string) {},
		"marker from an older hug rebase": func(t *testing.T, b *Repository, gitdir string) {
			// A hug rebase long finished left its marker (hug died after
			// git), then a person started their own.
			must(t, os.WriteFile(filepath.Join(gitdir, rebaseMarker), []byte("0000000000000000000000000000000000000000\n"), 0o600))
		},
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			a, b := pair(t)
			writeFile(t, a, "task.md", "alice\n")
			must(t, a.Sync())
			writeFile(t, b, "task.md", "bob\n")
			must(t, b.Commit(nil, "bob"))
			_, err := b.fetch()
			must(t, err)
			gitdir := command(t, b.Dir, "rev-parse", "--absolute-git-dir")
			setup(t, b, gitdir)
			c := exec.Command("git", "-C", b.Dir, "rebase", b.remoteRef())
			c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@x", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@x")
			if c.Run() == nil {
				t.Fatal("expected the person's rebase to stop on a conflict")
			}
			defer func() { _, _ = b.git("rebase", "--abort") }()
			must(t, b.Recover())
			var typed *Error
			if err = b.Sync(); !errors.As(err, &typed) || typed.Op != "worktree busy" || typed.Fix != b.fix("rebase --abort") {
				t.Fatalf("expected busy with fix %q: %v", b.fix("rebase --abort"), err)
			}
			if _, err = os.Stat(filepath.Join(gitdir, "rebase-merge", "head-name")); err != nil {
				t.Fatalf("the person's rebase was touched: %v", err)
			}
		})
	}
}

// A hug rebase still running (its process holds the lock) is never ended
// by another process's write.
func TestRunningHugRebaseIsLeftAlone(t *testing.T) {
	_, b, gitdir := stuckRebase(t, "pick")
	end, err := b.startRebase(strings.TrimSpace(command(t, b.Dir, "rev-parse", "refs/heads/huginn")))
	must(t, err)
	defer end(true)
	other := *b
	must(t, other.Recover())
	if _, err = os.Stat(filepath.Join(gitdir, "rebase-merge", "head-name")); err != nil {
		t.Fatalf("a running hug rebase was ended: %v", err)
	}
}

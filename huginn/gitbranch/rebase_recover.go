package gitbranch

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/veridit/hug/internal/flock"
)

// A rebase hug starts is marked (fcf4fg): before git rebase it writes
// hug-rebase (the head it rebases) in the worktree's git directory and holds
// hug-rebase.lock until the rebase has finished or been aborted. A process
// that dies (killed, crashed, the machine slept and was reaped) releases
// the lock with it, so a later write that finds rebase-merge, the marker and
// a free lock knows the rebase is hug's own and that nobody is running it:
// it aborts it, re-applies an autostash, and goes on. Without the marker
// (a person's or another tool's rebase) the write still stops with the fix.
// Seid, 2026-11-16: an interrupted sync left rebase-merge holding only an
// autostash and every agent's write failed until someone ran git rebase
// --quit and committed by hand.

const rebaseMarker = "hug-rebase"

// BeforeRebase, for tests only, runs just before hug's git rebase in dir:
// an editor writing in the window between hug's clean check and the rebase.
var BeforeRebase func(dir string)

// startRebase marks the rebase about to start and returns its end: keep
// leaves the marker for recoverRebase, after a rebase that failed.
func (r *Repository) startRebase(head string) (func(keep bool), error) {
	dir, _, e := gitDir(r.Dir)
	if e != nil {
		return nil, e
	}
	marker := filepath.Join(dir, rebaseMarker)
	release, e := flock.Acquire(marker + ".lock")
	if e != nil {
		return nil, e
	}
	if e = os.WriteFile(marker, []byte(head+"\n"), 0o600); e != nil {
		release()
		return nil, e
	}
	return func(keep bool) {
		if !keep {
			_ = os.Remove(marker)
		}
		release()
	}, nil
}

// startWindow is how long after hug's marker git may begin its rebase
// state for the rebase to count as hug's: a loaded Windows runner takes
// seconds to start git, never a minute.
const startWindow = time.Minute

// startedWith reports whether the rebase in state began with hug's marker
// written at marked: git writes these files once, at the start (the
// directory's own time moves with every pick).
func startedWith(state string, marked time.Time) bool {
	var first time.Time
	for _, name := range []string{"autostash", "head-name", "orig-head", "onto"} {
		if st, e := os.Stat(filepath.Join(state, name)); e == nil && (first.IsZero() || st.ModTime().Before(first)) {
			first = st.ModTime()
		}
	}
	return !first.IsZero() && !first.Before(marked.Add(-time.Second)) && first.Before(marked.Add(startWindow))
}

// recoverRebase ends an interrupted rebase hug itself started in gitdir and
// reports whether it did, with one line through OnRepaired when report is
// set. It never touches one hug cannot prove its own.
func (r *Repository) recoverRebase(gitdir string, report bool) (bool, error) {
	marker := filepath.Join(gitdir, rebaseMarker)
	mst, e := os.Stat(marker)
	if e != nil {
		return false, nil
	}
	release, ok, e := flock.TryAcquire(marker + ".lock")
	if e != nil || !ok {
		return false, nil // a hug rebase is running right now
	}
	defer release()
	state := ""
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if _, e := os.Stat(filepath.Join(gitdir, name)); e == nil {
			state = filepath.Join(gitdir, name)
		}
	}
	if state != "" && !startedWith(state, mst.ModTime()) {
		return false, nil
	}
	if state == "" {
		_ = os.Remove(marker) // hug died before git began: nothing to undo
		return false, nil
	}
	b, _ := os.ReadFile(marker)
	orig := strings.TrimSpace(string(b))
	if head, e := os.ReadFile(filepath.Join(state, "orig-head")); e == nil && strings.TrimSpace(string(head)) != orig {
		return false, nil
	}
	if name, e := os.ReadFile(filepath.Join(state, "head-name")); e == nil && strings.TrimSpace(string(name)) != "refs/heads/"+r.Branch {
		return false, nil
	}
	// The killed git's index lock, taken after hug's marker; any other
	// stops the write as before.
	lock := filepath.Join(gitdir, "index.lock")
	if st, e := os.Stat(lock); e == nil {
		if st.ModTime().Before(mst.ModTime().Add(-time.Second)) {
			return false, nil
		}
		if e = os.Remove(lock); e != nil {
			return false, r.ioerr(e)
		}
	}
	stash := ""
	if b, e := os.ReadFile(filepath.Join(state, "autostash")); e == nil {
		stash = strings.TrimSpace(string(b))
	}
	// Ending the rebase resets the files. A change there that neither the
	// autostash nor the rebase itself made (a write by a hug without this
	// repair, a person's edit) is kept in the stash first.
	saved, e := r.saveForeign(state, stash, orig)
	if e != nil {
		return false, e
	}
	if _, e := os.Stat(filepath.Join(state, "head-name")); e == nil {
		// A rebase with its full state: git's own abort puts the branch
		// back where it was and re-applies the autostash.
		if _, e = r.git("rebase", "--abort"); e != nil {
			return false, e
		}
	} else {
		// Stopped before git wrote more than the autostash (Seid's case):
		// git rebase --abort refuses this state. The branch must still be
		// where hug's rebase found it; then put HEAD back on it, the files
		// back to it, and the stashed edits on top.
		if at, e := r.git("rev-parse", "--verify", "--quiet", "refs/heads/"+r.Branch); e != nil || strings.TrimSpace(at) != orig {
			return false, nil
		}
		if e := os.RemoveAll(state); e != nil {
			return false, r.ioerr(e)
		}
		if _, e := r.git("symbolic-ref", "HEAD", "refs/heads/"+r.Branch); e != nil {
			return false, e
		}
		if _, e := r.git("reset", "--hard", "HEAD"); e != nil {
			return false, e
		}
		if stash != "" {
			if _, e := r.git("stash", "apply", stash); e != nil {
				_, _ = r.git("reset", "--hard", "HEAD")
				_, _ = r.git("stash", "store", "-m", "hug: edits from an interrupted sync", stash)
				_ = os.Remove(marker)
				return false, &Error{Op: "sync recovery", Detail: "ended an interrupted hug rebase, but its stashed edits do not apply on " + r.Branch + "; they are kept in the stash", Fix: r.fix("stash pop")}
			}
		}
	}
	_ = os.Remove(marker)
	line := "ended an interrupted hug rebase in " + r.Dir + "; local commits kept"
	if stash != "" {
		line += ", stashed edits re-applied"
	}
	if saved {
		line += "; the files as they were are in git stash list"
	}
	if report && r.OnRepaired != nil {
		r.OnRepaired(line)
	}
	return true, nil
}

// Recover ends an interrupted rebase hug itself started. Callers hold the
// board turn and call it before reading the board, never after writing to
// it: ending the rebase resets the worktree to the branch (fcf4fg). A
// rebase hug cannot prove its own is left alone; the write's own check
// then stops with its fix line, as before.
func (r *Repository) Recover() error {
	dir, _, e := gitDir(r.Dir)
	if e != nil {
		return nil
	}
	for _, name := range []string{"rebase-merge", "rebase-apply"} {
		if _, e := os.Stat(filepath.Join(dir, name)); e == nil {
			_, e = r.recoverRebase(dir, true)
			return e
		}
	}
	// A marker without a rebase: hug died after git finished. Nothing to
	// undo; drop it so it can never vouch for a later rebase.
	marker := filepath.Join(dir, rebaseMarker)
	if _, e := os.Stat(marker); e == nil {
		if release, ok, _ := flock.TryAcquire(marker + ".lock"); ok {
			_ = os.Remove(marker)
			release()
		}
	}
	return nil
}

// saveForeign stores the worktree in the stash when it changed a path the
// interrupted rebase had no business with: not in its autostash, not
// between its start and where it stopped.
func (r *Repository) saveForeign(state, stash, orig string) (bool, error) {
	status, e := r.git("status", "--porcelain", "-z")
	if e != nil || status == "" {
		return false, e
	}
	ours := map[string]bool{}
	add := func(out string) {
		for _, p := range strings.Split(out, "\x00") {
			if p != "" {
				ours[p] = true
			}
		}
	}
	if stash != "" {
		out, _ := r.git("diff", "--name-only", "-z", stash+"^1", stash)
		add(out)
	}
	out, _ := r.git("diff", "--name-only", "-z", "HEAD", orig)
	add(out)
	foreign := false
	fields := strings.Split(status, "\x00")
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if len(f) < 4 {
			continue
		}
		if f[0] == 'R' || f[0] == 'C' {
			i++ // the rename's source follows
		}
		if !ours[f[3:]] {
			foreign = true
		}
	}
	if !foreign {
		return false, nil
	}
	oid, e := r.git("stash", "create", "hug: worktree when an interrupted sync was ended")
	if e != nil || strings.TrimSpace(oid) == "" {
		return false, &Error{Op: "sync recovery", Detail: "an interrupted hug rebase holds edits it did not make, and they cannot be stashed; nothing was changed", Fix: r.fix("status")}
	}
	if _, e = r.git("stash", "store", "-m", "hug: worktree when an interrupted sync was ended", strings.TrimSpace(oid)); e != nil {
		return false, e
	}
	return true, nil
}

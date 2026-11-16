package gitbranch

import (
	"path/filepath"
	"strings"
)

// Place is where in a repository work happens: the branch checked out in
// dir's worktree, that worktree's folder, and the main worktree's folder
// (x1t1t1). The branch is what matters; the folder only says where to look now.
type Place struct {
	Branch   string // short name ("feat/x"); empty when detached
	Worktree string // top level of dir's worktree
	Main     string // top level of the main worktree
	Linked   bool   // dir is in a linked worktree (git worktree add), not the main one
}

// Where reports the Place of dir. Outside a git repository it returns an error.
func Where(dir string) (Place, error) {
	r := &Repository{Dir: dir}
	top, err := r.runAt(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Place{}, err
	}
	p := Place{Worktree: filepath.Clean(top)}
	if b, err := r.runAt(dir, "symbolic-ref", "--quiet", "--short", "HEAD"); err == nil {
		p.Branch = strings.TrimSpace(b)
	}
	list, err := r.worktrees(p.Worktree)
	if err != nil {
		return Place{}, err
	}
	p.Main = mainRoot(list, p.Worktree)
	p.Linked = !samePath(p.Main, p.Worktree)
	return p, nil
}

// DefaultBranch is the repository's main line at dir: what origin/HEAD
// points to, else a local main or master. Empty when none is known.
func DefaultBranch(dir string) string {
	r := &Repository{Dir: dir}
	if ref, err := r.runAt(dir, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if _, b, ok := strings.Cut(strings.TrimSpace(ref), "/"); ok && b != "" {
			return b
		}
	}
	for _, b := range []string{"main", "master"} {
		if _, err := r.runAt(dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+b); err == nil {
			return b
		}
	}
	return ""
}

func samePath(a, b string) bool {
	if pa, err := filepath.EvalSymlinks(a); err == nil {
		a = pa
	}
	if pb, err := filepath.EvalSymlinks(b); err == nil {
		b = pb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

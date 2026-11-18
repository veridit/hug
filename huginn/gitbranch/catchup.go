package gitbranch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Catching up across a layout move (w40cmk check 9, crab 04:44). This
// clone has unpushed commits made on the old layout and the remote moved
// the board (or both moved, each on its own). A rebase would replay the
// old commits' renames and edits against files that are no longer there.
// CatchUp replays them instead, one by one, onto the remote: each commit's
// change is read as files on the new layout (Translate), merged file by
// file with what the remote has (Merge), and committed with its own
// message and author. This clone's own layout move is not replayed: the
// remote's stands. Nothing is pushed; the next sync pushes the result.
//
// Every commit lands or none does: a file Merge cannot read stops the run,
// HEAD and the work tree are put back, and the error names the commit and
// the file.

// Translator reads the board on the remote's layout.
type Translator struct {
	// Names maps every path of a tree to its name on the new layout.
	Names func(paths []string) map[string]string
	// Content is a file's text on the new layout, from its old path and
	// text.
	Content func(path string, b []byte) []byte
	// Merge merges one file this commit changed. base or theirs is nil
	// when that side has no such file; it returns the content to write
	// and one line per value it kept from ours in a note.
	Merge func(path string, base, ours, theirs []byte, c Replayed) ([]byte, []string, error)
	// Layout reads a hug.toml's layout, as Repository.Layout.
	Layout func(text string) (int, error)
}

// Replayed is one local commit and what its replay did.
type Replayed struct {
	Commit, Subject, Author, Email, At string
	// Kept is one line per value both sides changed (the board's stays,
	// this clone's is in a note on the task).
	Kept []string
	// Skipped is set on this clone's own layout move, which is not
	// replayed: the remote's stands.
	Skipped bool
}

// CatchUpError is a replay that stopped; nothing changed.
type CatchUpError struct {
	Commit, Subject, Path, Detail, Fix string
	Kept                               int // local commits, all still there
}

func (e *CatchUpError) Error() string {
	return fmt.Sprintf("catch-up stopped at %s %q: %s: %s; HEAD is back where it was, %d local commits kept, nothing pushed\nfix: %s", e.Commit, e.Subject, e.Path, e.Detail, e.Kept, e.Fix)
}

// Pending lists the commits CatchUp would replay, oldest first, with the
// layout move marked Skipped; empty when the remote and this clone are on
// one layout or nothing is unpushed. It fetches nothing.
func (r *Repository) Pending(t Translator) ([]Replayed, error) {
	if r.Remote == "" {
		return nil, nil
	}
	if _, e := r.git("rev-parse", "--verify", "--quiet", r.remoteRef()); e != nil {
		return nil, nil
	}
	saved := r.Layout
	r.Layout = t.Layout
	e := r.sameLayout()
	r.Layout = saved
	if e == nil {
		return nil, nil
	}
	if _, ok := e.(*LayoutError); !ok {
		return nil, e
	}
	out, e := r.git("rev-list", "--reverse", "--topo-order", "--no-merges", "--format=%H%x00%s%x00%an%x00%ae%x00%aI", r.remoteRef()+"..HEAD")
	if e != nil {
		return nil, e
	}
	var list []Replayed
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, "commit ") || l == "" {
			continue
		}
		f := strings.SplitN(l, "\x00", 5)
		if len(f) != 5 {
			continue
		}
		c := Replayed{Commit: f[0], Subject: f[1], Author: f[2], Email: f[3], At: f[4]}
		moved, e := r.movesLayout(c.Commit, t)
		if e != nil {
			return nil, e
		}
		c.Skipped = moved
		list = append(list, c)
	}
	return list, nil
}

// movesLayout says whether commit changes hug.toml's layout.
func (r *Repository) movesLayout(commit string, t Translator) (bool, error) {
	parent, e := r.git("rev-parse", "--verify", "--quiet", commit+"^")
	if e != nil {
		return false, nil // a root commit moves nothing
	}
	saved := r.Layout
	r.Layout = t.Layout
	defer func() { r.Layout = saved }()
	now, e := r.layoutAt(commit)
	if e != nil {
		return false, e
	}
	then, e := r.layoutAt(parent)
	return now != then, e
}

// tree is a commit's files on the new layout: new path to the old path
// and blob id, so two trees compare without reading a file.
type tree map[string]struct{ old, blob string }

func (r *Repository) tree(commit string, t Translator) (tree, error) {
	out, e := r.git("ls-tree", "-r", "-z", "--full-tree", "--end-of-options", commit+"^{tree}")
	if e != nil {
		return nil, e
	}
	blobs := map[string]string{}
	var paths []string
	for _, l := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		// "<mode> blob <id>\t<path>"
		meta, p, ok := strings.Cut(l, "\t")
		if !ok {
			continue
		}
		f := strings.Fields(meta)
		if len(f) != 3 || f[1] != "blob" {
			continue
		}
		blobs[p] = f[2]
		paths = append(paths, p)
	}
	tr := tree{}
	for old, to := range t.Names(paths) {
		tr[to] = struct{ old, blob string }{old, blobs[old]}
	}
	return tr, nil
}

// blob is a file's bytes, exactly (git() trims a trailing newline).
func (r *Repository) blob(id string) ([]byte, error) {
	c := exec.Command("git", "-C", r.Dir, "cat-file", "blob", id)
	c.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	b, e := c.Output()
	if e != nil {
		return nil, &Error{Op: "cat-file blob " + id, Detail: e.Error(), Fix: r.fix("cat-file -t " + id)}
	}
	return b, nil
}

// file is a tree's file on the new layout; nil when it has none.
func (r *Repository) file(tr tree, p string, t Translator) ([]byte, error) {
	f, ok := tr[p]
	if !ok {
		return nil, nil
	}
	b, e := r.blob(f.blob)
	if e != nil {
		return nil, e
	}
	return t.Content(f.old, b), nil
}

// CatchUp replays this clone's unpushed commits onto the fetched remote
// (see above). It returns what each commit did, oldest first. It fetches
// first, so it replays onto the board as it is now.
func (r *Repository) CatchUp(t Translator) ([]Replayed, error) {
	if e := r.checkIdle(); e != nil {
		return nil, e
	}
	if r.Remote != "" {
		if _, e := r.fetch(); e != nil {
			return nil, e
		}
	}
	if status, e := r.git("status", "--porcelain"); e != nil {
		return nil, e
	} else if status != "" {
		return nil, &Error{Op: "catch-up", Detail: "worktree has uncommitted changes", Fix: r.fix("add --all") + " && " + r.fix("commit -m "+fmt.Sprintf("%q", "Save local changes"))}
	}
	list, e := r.Pending(t)
	if e != nil || len(list) == 0 {
		return list, e
	}
	head, e := r.Head()
	if e != nil {
		return nil, e
	}
	stop := func(c Replayed, path string, cause error) error {
		// Put HEAD and the work tree back: every local commit is still
		// there, as it was.
		_, _ = r.git("reset", "-q", "--hard", head)
		detail := strings.SplitN(cause.Error(), "\nfix: ", 2)[0]
		// The file as that commit has it: what to read before making the
		// change again with hug task (the only stop is a file hug cannot
		// read, which hug itself never writes).
		fix := r.fix("show " + c.Commit[:12] + " -- " + path)
		return &CatchUpError{Commit: c.Commit[:12], Subject: c.Subject, Path: path, Detail: detail, Fix: fix, Kept: len(list)}
	}
	if _, e = r.git("reset", "-q", "--hard", r.remoteRef()); e != nil {
		_, _ = r.git("reset", "-q", "--hard", head)
		return nil, e
	}
	for i := range list {
		c := &list[i]
		if c.Skipped {
			continue
		}
		parent, e := r.git("rev-parse", "--verify", c.Commit+"^")
		if e != nil {
			return nil, stop(*c, "(root commit)", e)
		}
		before, e := r.tree(parent, t)
		if e != nil {
			return nil, stop(*c, "its parent", e)
		}
		after, e := r.tree(c.Commit, t)
		if e != nil {
			return nil, stop(*c, "its tree", e)
		}
		// A file changed when its blob or its old name did (a rank move
		// on the old layout is a rename: on the new one, its rank: line).
		var touched []string
		for p, v := range after {
			if old, ok := before[p]; !ok || old != v {
				touched = append(touched, p)
			}
		}
		for p := range before {
			if _, ok := after[p]; !ok {
				touched = append(touched, p)
			}
		}
		sort.Strings(touched)
		for _, p := range touched {
			disk := filepath.Join(r.Dir, filepath.FromSlash(p))
			theirs, e := os.ReadFile(disk)
			if os.IsNotExist(e) {
				theirs, e = nil, nil
			}
			if e != nil {
				return nil, stop(*c, p, e)
			}
			base, e := r.file(before, p, t)
			if e != nil {
				return nil, stop(*c, p, e)
			}
			if _, inOurs := after[p]; !inOurs {
				// Removed here: removed there too, unless the board
				// changed it since (then its change stays).
				if theirs != nil && string(theirs) == string(base) {
					if e := os.Remove(disk); e != nil {
						return nil, stop(*c, p, e)
					}
				}
				continue
			}
			ours, e := r.file(after, p, t)
			if e != nil {
				return nil, stop(*c, p, e)
			}
			merged, kept, e := t.Merge(p, base, ours, theirs, *c)
			if e != nil {
				return nil, stop(*c, p, e)
			}
			c.Kept = append(c.Kept, kept...)
			if e := os.MkdirAll(filepath.Dir(disk), 0o755); e != nil {
				return nil, stop(*c, p, e)
			}
			if e := os.WriteFile(disk, merged, 0o644); e != nil {
				return nil, stop(*c, p, e)
			}
		}
		if _, e := r.git("add", "--all", "--", "."); e != nil {
			return nil, stop(*c, "(staging)", e)
		}
		if staged, _ := r.git("diff", "--cached", "--name-only"); staged == "" {
			continue // already on the board: nothing left to replay
		}
		// The commit's own message and author; the committer is this run.
		if _, e := r.git("commit", "-q", "--no-verify", "-C", c.Commit); e != nil {
			return nil, stop(*c, "(commit)", e)
		}
	}
	return list, nil
}

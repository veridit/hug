package gitbranch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Catching up across a layout move (w40cmk check 9, crab 04:44). This
// clone has unpushed commits made on the old layout and the remote moved
// the board (or this clone moved first). A rebase would replay the
// old commits' renames and edits against files that are no longer there.
// CatchUp replays them instead, one by one, onto the remote: each commit's
// change is read as files on the new layout (Translate), merged file by
// file with what the remote has (Merge), and committed with its own
// message and author. A pure layout move is omitted only if the translated
// base is already committed. Nothing is pushed; the next sync pushes the result.
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
	Content func(path, translated string, b []byte) []byte
	// Merge merges one file this commit changed. base or theirs is nil
	// when that side has no such file; it returns the content to write
	// and one line per value it kept from ours in a note.
	Merge func(path string, base, ours, theirs []byte, c Replayed) ([]byte, []string, error)
	// Recover marks a non-task true conflict. The board's active path stays;
	// this commit's exact original blob is saved at path.local-FULLCOMMIT.
	Recover func(path string, base, ours, theirs []byte) bool
	// Layout reads a hug.toml's layout, as Repository.Layout.
	Layout func(text string) (int, error)
}

// Replayed is one local commit and what its replay did.
type Replayed struct {
	Commit, Subject, Author, Email, At string
	// Kept is one line per value both sides changed (the board's stays,
	// this clone's is in a note on the task).
	Kept []string
	// Skipped marks a pure layout move. It is omitted only if no staged
	// translation remains, otherwise its message retains that translation.
	Skipped bool
}

// CatchUpError is a replay that stopped; nothing changed.
type CatchUpError struct {
	Commit, Subject, Path, Detail, Fix string
	Operation                          string // empty means upgrade catch-up
	Kept                               int    // local commits, all still there
}

func (e *CatchUpError) Error() string {
	op := e.Operation
	if op == "" {
		op = "catch-up"
	}
	return fmt.Sprintf("%s stopped at %s %q: %s: %s; HEAD is back where it was, %d local commits kept, nothing pushed\nfix: %s", op, e.Commit, e.Subject, e.Path, e.Detail, e.Kept, e.Fix)
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
	probe := *r
	probe.Layout = t.Layout
	e := probe.sameLayout()
	if e == nil {
		return nil, nil
	}
	if _, ok := e.(*LayoutError); !ok {
		return nil, e
	}
	return r.pendingCommits(t)
}

// pendingCommits reads complete first-parent deltas for either kind of
// replay. The caller decides whether cross-layout recovery was requested
// or an ordinary same-layout sync found divergent commits.
func (r *Repository) pendingCommits(t Translator) ([]Replayed, error) {
	out, e := r.git("rev-list", "--reverse", "--first-parent", "--format=%H%x00%s%x00%an%x00%ae%x00%aI", r.remoteRef()+"..HEAD")
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
			return nil, &Error{Op: "catch-up preview", Detail: "cannot read a local commit's metadata; no commit was skipped", Fix: r.fix("log --oneline")}
		}
		c := Replayed{Commit: f[0], Subject: f[1], Author: f[2], Email: f[3], At: f[4]}
		moved, e := r.movesLayout(c.Commit, t)
		if e != nil {
			return nil, e
		}
		if moved {
			before, e := r.tree(c.Commit+"^", t)
			if e != nil {
				return nil, e
			}
			after, e := r.tree(c.Commit, t)
			if e != nil {
				return nil, e
			}
			c.Skipped, e = r.sameTranslated(before, after, t)
			if e != nil {
				return nil, e
			}
		}
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
	probe := *r
	probe.Layout = t.Layout
	now, e := probe.layoutAt(commit)
	if e != nil {
		return false, e
	}
	then, e := probe.layoutAt(parent)
	return now != then, e
}

func (r *Repository) sameTranslated(a, b tree, t Translator) (bool, error) {
	if len(a) != len(b) {
		return false, nil
	}
	for p, av := range a {
		bv, ok := b[p]
		if !ok || av.mode != bv.mode {
			return false, nil
		}
		x, e := r.file(a, p, t)
		if e != nil {
			return false, e
		}
		y, e := r.file(b, p, t)
		if e != nil {
			return false, e
		}
		if string(x) != string(y) {
			return false, nil
		}
	}
	return true, nil
}

// tree is a commit's files on the new layout: new path to the old path
// and blob id, so two trees compare without reading a file.
type tree map[string]struct{ old, blob, mode string }

type replayPathError struct {
	path, detail, fix string
}

func (e *replayPathError) Error() string { return e.path + ": " + e.detail + "\nfix: " + e.fix }

func replayMode(base, ours, theirs string) string {
	switch {
	case theirs == "":
		return ours
	case ours == base:
		return theirs
	case theirs == base, ours == theirs:
		return ours
	default:
		return theirs
	}
}

func (r *Repository) tree(commit string, t Translator) (tree, error) {
	bad := func(p, detail string) error {
		return &replayPathError{p, detail, r.fix("show " + quote(commit) + " -- " + quote(p))}
	}
	out, e := r.git("ls-tree", "-r", "-z", "--full-tree", "--end-of-options", commit+"^{tree}")
	if e != nil {
		return nil, e
	}
	blobs := map[string]string{}
	modes := map[string]string{}
	var paths []string
	for _, l := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
		// "<mode> blob <id>\t<path>"
		meta, p, ok := strings.Cut(l, "\t")
		if !ok {
			continue
		}
		f := strings.Fields(meta)
		if len(f) != 3 {
			return nil, bad(p, "hug cannot parse this tree entry")
		}
		if f[1] != "blob" || (f[0] != "100644" && f[0] != "100755") {
			return nil, bad(p, "hug cannot replay this file as a regular board file (mode "+f[0]+")")
		}
		blobs[p] = f[2]
		modes[p] = f[0]
		paths = append(paths, p)
	}
	tr := tree{}
	names := t.Names(paths)
	for _, old := range paths {
		to, ok := names[old]
		if !ok || to == "" || filepath.IsAbs(to) || filepath.ToSlash(filepath.Clean(to)) != to || strings.HasPrefix(to, "../") || strings.Contains(to, "\\") {
			return nil, bad(old, "hug cannot read the translated path "+fmt.Sprintf("%q", to))
		}
		if prior, exists := tr[to]; exists {
			return nil, bad(old, "both "+prior.old+" and "+old+" translate to "+to+"; neither file was discarded")
		}
		tr[to] = struct{ old, blob, mode string }{old, blobs[old], modes[old]}
	}
	return tr, nil
}

// blob is a file's bytes, exactly (git() trims a trailing newline).
func (r *Repository) blob(id string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	c := exec.CommandContext(ctx, "git", "-C", r.Dir, "cat-file", "blob", id)
	c.WaitDelay = 2 * time.Second
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
	return t.Content(f.old, p, b), nil
}

// prepareReplay uses the same name/content translator as each commit, not
// a second migration algorithm. Remote layout2 may be the base when this
// clone independently moved first. Materialize its coherent translated tree
// privately and stage it for the first replay commit, never on live HEAD.
func (r *Repository) prepareReplay(commit string, tr tree, t Translator) error {
	needed := false
	for p, f := range tr {
		needed = needed || p != f.old
	}
	if config, ok := tr["hug.toml"]; ok {
		raw, e := r.blob(config.blob)
		if e != nil {
			return e
		}
		needed = needed || !bytes.Equal(raw, t.Content(config.old, "hug.toml", raw))
	}
	if !needed {
		return nil
	}
	var paths []string
	for p := range tr {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	bad := func(p string, e error) error {
		original := tr[p].old
		return &replayPathError{original, e.Error(), r.fix("show " + quote(commit+":"+original))}
	}
	// Validate every translated file before any filesystem transform. A
	// selected task is parsed by the same resolver used during replay.
	content := map[string][]byte{}
	for _, p := range paths {
		raw, e := r.file(tr, p, t)
		if e != nil {
			return bad(p, e)
		}
		validated, _, e := t.Merge(p, nil, raw, raw, Replayed{})
		if e != nil {
			return bad(p, e)
		}
		content[p] = validated
	}
	for _, p := range paths {
		if old := tr[p].old; old != p {
			if e := os.Remove(filepath.Join(r.Dir, filepath.FromSlash(old))); e != nil {
				return bad(p, e)
			}
		}
	}
	for _, p := range paths {
		disk := filepath.Join(r.Dir, filepath.FromSlash(p))
		if e := os.MkdirAll(filepath.Dir(disk), 0o755); e != nil {
			return bad(p, e)
		}
		mode := os.FileMode(0o644)
		if tr[p].mode == "100755" {
			mode = 0o755
		}
		if e := os.WriteFile(disk, content[p], mode); e != nil {
			return bad(p, e)
		}
		if e := os.Chmod(disk, mode); e != nil {
			return bad(p, e)
		}
	}
	if _, e := r.git("add", "--all", "--", "."); e != nil {
		return e
	}
	for _, p := range paths {
		chmod := "--chmod=-x"
		if tr[p].mode == "100755" {
			chmod = "--chmod=+x"
		}
		if _, e := r.git("update-index", chmod, "--", p); e != nil {
			return bad(p, e)
		}
	}
	return nil
}

// CatchUp replays this clone's unpushed commits onto the fetched remote
// (see above). It returns what each commit did, oldest first. It fetches
// only for known layout divergence, then replays onto the fetched board.
func (r *Repository) CatchUp(t Translator) (replayed []Replayed, err error) {
	if e := r.checkIdle(); e != nil {
		return nil, e
	}
	// Ordinary migration is local and works offline. Only a known layout
	// divergence warrants fetching and replaying remote state.
	list, e := r.Pending(t)
	if e != nil || len(list) == 0 {
		return list, e
	}
	if r.Remote != "" {
		if _, e := r.fetch(); e != nil {
			return nil, &Error{Op: "catch-up", Detail: "remote could not be fetched; HEAD and local commits are unchanged", Fix: "hug upgrade --apply", Cause: e}
		}
	}
	if status, e := r.git("status", "--porcelain"); e != nil {
		return nil, e
	} else if status != "" {
		return nil, &Error{Op: "catch-up", Detail: "worktree has uncommitted changes", Fix: r.fix("add --all") + " && " + r.fix("commit -m "+fmt.Sprintf("%q", "Save local changes"))}
	}
	list, e = r.Pending(t)
	if e != nil || len(list) == 0 {
		return list, e
	}
	return r.replay(list, t)
}

// resolveDiverged uses semantic replay for ordinary same-layout sync too.
// It does not fetch and never makes upgrade/catch-up a prerequisite. Even
// a text merge without Git conflicts can mix two edits of one scalar
// section, so every divergent task-board merge uses the shared resolver.
func (r *Repository) resolveDiverged() (bool, error) {
	if r.Resolver == nil {
		return false, nil
	}
	counts, e := r.git("rev-list", "--left-right", "--count", "HEAD..."+r.remoteRef())
	if e != nil {
		return true, e
	}
	f := strings.Fields(counts)
	if len(f) != 2 {
		return true, &Error{Op: "sync", Detail: "cannot read divergent commit counts; nothing replayed", Fix: r.fix("status")}
	}
	left, le := strconv.Atoi(f[0])
	right, re := strconv.Atoi(f[1])
	if le != nil || re != nil {
		return true, &Error{Op: "sync", Detail: "cannot read divergent commit counts; nothing replayed", Fix: r.fix("status")}
	}
	if left == 0 || right == 0 {
		return false, nil // normal fast-forward or already ahead
	}
	list, e := r.pendingCommits(*r.Resolver)
	if e != nil {
		return true, e
	}
	list, e = r.replay(list, *r.Resolver)
	if e != nil {
		var stopped *CatchUpError
		if errors.As(e, &stopped) {
			stopped.Operation = "sync"
		}
		return true, e
	}
	if r.OnResolved != nil {
		for _, c := range list {
			r.OnResolved(c)
		}
	}
	return true, nil
}

// replay validates and materializes the complete result privately before
// replacing live HEAD. Both ordinary sync and layout catch-up use it.
func (r *Repository) replay(list []Replayed, t Translator) (replayed []Replayed, err error) {
	if e := r.checkIdle(); e != nil {
		return nil, e
	}
	if status, e := r.git("status", "--porcelain"); e != nil {
		return nil, e
	} else if status != "" {
		return nil, &Error{Op: "sync replay", Detail: "worktree has uncommitted changes; no replay installed", Fix: r.fix("status")}
	}
	head, e := r.Head()
	if e != nil {
		return nil, e
	}
	// Never replay against the live board. A failure after writing a new
	// path must not leave untracked files behind, and an interrupted run
	// must not strand HEAD halfway through the replay.
	live := r
	// Validate remote paths too, before checking out or reading their
	// contents. In particular, never follow a remote symlink while writing
	// a local attachment beneath it.
	remoteTree, e := live.tree(live.remoteRef(), t)
	if e != nil {
		return nil, e
	}
	dir, e := os.MkdirTemp("", "hug-catchup-")
	if e != nil {
		return nil, e
	}
	if _, e = live.git("-c", "core.hooksPath="+os.DevNull, "worktree", "add", "--detach", dir, live.remoteRef()); e != nil {
		_ = os.Remove(dir)
		return nil, e
	}
	defer func() {
		if _, e := live.git("worktree", "remove", "--force", dir); e != nil {
			err = errors.Join(err, e)
		}
	}()
	private := *live
	private.Dir = dir
	r = &private
	if e := r.prepareReplay(live.remoteRef(), remoteTree, t); e != nil {
		var pe *replayPathError
		if errors.As(e, &pe) {
			return nil, &CatchUpError{Commit: live.remoteRef(), Subject: "remote board translation", Path: pe.path, Detail: pe.detail, Fix: pe.fix, Kept: len(list)}
		}
		return nil, e
	}
	sources := map[string]string{}
	stop := func(c Replayed, path string, cause error) error {
		// The live board has not changed: every local commit is still
		// there, as it was. The private worktree is removed on return.
		detail := strings.SplitN(cause.Error(), "\nfix: ", 2)[0]
		var pe *replayPathError
		if errors.As(cause, &pe) {
			path, detail = pe.path, pe.detail
		}
		// The file as that commit has it: what to read before making the
		// change again with hug task (the only stop is a file hug cannot
		// read, which hug itself never writes).
		original := path
		if old, ok := sources[path]; ok {
			original = old
		}
		fix := live.fix("show " + quote(c.Commit+":"+original))
		return &CatchUpError{Commit: c.Commit[:12], Subject: c.Subject, Path: path, Detail: detail, Fix: fix, Kept: len(list)}
	}
	for i := range list {
		c := &list[i]
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
		current, e := r.tree("HEAD", t)
		if e != nil {
			return nil, stop(*c, "the board's tree", e)
		}
		sources = map[string]string{}
		for p, f := range after {
			sources[p] = f.old
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
		modeChanges := map[string]string{}
		for _, p := range touched {
			disk := filepath.Join(r.Dir, filepath.FromSlash(p))
			theirs, e := r.file(current, p, t)
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
			if t.Recover != nil && t.Recover(p, base, ours, theirs) {
				recovery := p + ".local-" + c.Commit
				sources[recovery] = after[p].old
				original, e := r.blob(after[p].blob)
				if e != nil {
					return nil, stop(*c, p, e)
				}
				if prior, exists := current[recovery]; exists {
					priorBytes, e := r.blob(prior.blob)
					if e != nil {
						return nil, stop(*c, recovery, e)
					}
					if !bytes.Equal(priorBytes, original) || prior.mode != after[p].mode {
						return nil, stop(*c, recovery, fmt.Errorf("recovery collision: %s already has different content or mode; neither copy was overwritten", recovery))
					}
				} else {
					// Another source path in this commit must not overwrite a
					// freshly generated recovery later in the replay loop.
					if _, exists := after[recovery]; exists {
						return nil, stop(*c, recovery, fmt.Errorf("recovery collision: %s also exists in the local commit", recovery))
					}
					recoveryDisk := filepath.Join(r.Dir, filepath.FromSlash(recovery))
					if e := os.MkdirAll(filepath.Dir(recoveryDisk), 0o755); e != nil {
						return nil, stop(*c, recovery, e)
					}
					mode := os.FileMode(0o644)
					if after[p].mode == "100755" {
						mode = 0o755
					}
					if e := os.WriteFile(recoveryDisk, original, mode); e != nil {
						return nil, stop(*c, recovery, e)
					}
					if after[p].mode == "100755" {
						modeChanges[recovery] = "100755"
					}
				}
				c.Kept = append(c.Kept, fmt.Sprintf("%s: kept the board's bytes at %s; exact local bytes and mode from commit %s retained at %s", p, p, c.Commit, recovery))
			}
			merged, kept, e := t.Merge(p, base, ours, theirs, *c)
			if e != nil {
				return nil, stop(*c, p, e)
			}
			c.Kept = append(c.Kept, kept...)
			if merged == nil {
				if e := os.Remove(disk); e != nil && !os.IsNotExist(e) {
					return nil, stop(*c, p, e)
				}
				continue
			}
			if e := os.MkdirAll(filepath.Dir(disk), 0o755); e != nil {
				return nil, stop(*c, p, e)
			}
			if e := os.WriteFile(disk, merged, 0o644); e != nil {
				return nil, stop(*c, p, e)
			}
			chosen := replayMode(before[p].mode, after[p].mode, current[p].mode)
			mode := os.FileMode(0o644)
			if chosen == "100755" {
				mode = 0o755
			}
			if e := os.Chmod(disk, mode); e != nil {
				return nil, stop(*c, p, e)
			}
			if chosen != current[p].mode && (current[p].mode != "" || chosen == "100755") {
				modeChanges[p] = chosen
			}
		}
		if _, e := r.git("add", "--all", "--", "."); e != nil {
			return nil, stop(*c, "(staging)", e)
		}
		var modePaths []string
		for p := range modeChanges {
			modePaths = append(modePaths, p)
		}
		sort.Strings(modePaths)
		for _, p := range modePaths {
			if mode, ok := modeChanges[p]; ok {
				chmod := "--chmod=-x"
				if mode == "100755" {
					chmod = "--chmod=+x"
				}
				if _, e := r.git("update-index", chmod, "--", p); e != nil {
					return nil, stop(*c, p, e)
				}
			}
		}
		staged, e := r.git("diff", "--cached", "--name-only")
		if e != nil {
			return nil, stop(*c, "(staging)", e)
		}
		if staged == "" && c.Skipped {
			continue // only a layout move, already on the remote
		}
		// The commit's own message and author; the committer is this run.
		if _, e := r.git("commit", "-q", "--no-verify", "--allow-empty", "-C", c.Commit); e != nil {
			return nil, stop(*c, "(commit)", e)
		}
		// A layout-changing commit may contain other edits. Translate
		// and compare it just like every other commit, never skip its
		// entire delta merely because it also bumped layout.
		c.Skipped = false
	}
	// A translated base is never installed as an uncommitted transform,
	// even if all source commits were pure layout moves.
	if staged, e := r.git("diff", "--cached", "--name-only"); e != nil {
		return nil, e
	} else if staged != "" {
		if _, e := r.git("commit", "-q", "--no-verify", "-m", "hug upgrade: translate the remote board to layout 3"); e != nil {
			return nil, e
		}
	}
	result, e := r.Head()
	if e != nil {
		return nil, e
	}
	if now, e := live.Head(); e != nil || now != head {
		return nil, &Error{Op: "catch-up", Detail: "board HEAD changed while replaying; no replay was installed", Fix: "hug upgrade --apply", Cause: e}
	}
	if status, e := live.git("status", "--porcelain"); e != nil || status != "" {
		return nil, &Error{Op: "catch-up", Detail: "board acquired uncommitted changes while replaying; no replay was installed", Fix: live.fix("status"), Cause: e}
	}
	if _, e := live.git("reset", "--hard", result); e != nil {
		return nil, e
	}
	return list, nil
}

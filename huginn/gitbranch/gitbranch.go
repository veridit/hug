// Package gitbranch stores independent data on a parameterized orphan branch.
// It shells out to Git >= 2.42. With no remote it operates locally: Sync and
// Update still commit, but cannot provide cross-clone exclusion. A Repository
// must not be used concurrently. Separate clones coordinate through push.
package gitbranch

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Identity overrides Git's configured author and committer for this handle.
type Identity struct{ Name, Email string }
type Option func(*Repository)

func WithIdentity(name, email string) Option {
	return func(r *Repository) { r.Identity = Identity{name, email} }
}

// Repository is a branch worktree. Dir is absolute. Remote is empty in local-only mode.
// Root is the checkout Open was called from. In a linked worktree (git
// worktree add) Dir is the branch's one checkout, in the main worktree.
type Repository struct {
	Root, Dir, Branch, Remote string
	Common                    string // absolute common git dir, set by Open
	Identity                  Identity
}

// Error includes a copyable repair command and the original Git diagnostic.
type Error struct {
	Op, Detail, Fix string
	Cause           error
}

func (e *Error) Error() string { return e.Op + ": " + e.Detail + "\nfix: " + e.Fix }
func (e *Error) Unwrap() error { return e.Cause }

// ConflictError reports files that conflicted. The rebase has been aborted.
type ConflictError struct {
	Files []string
	Fix   string
}

func (e *ConflictError) Error() string {
	return "sync conflicts: " + strings.Join(e.Files, ", ") + " (rebase aborted)\nfix: " + e.Fix
}

// ErrRaced means another clone won the push. Dir has been reset to RemoteHead,
// so the caller can read the winner's files before reporting its assignee.
type ErrRaced struct{ RemoteHead, Dir, Fix string }

func (e *ErrRaced) Error() string {
	return "update lost push race; remote state restored at " + e.RemoteHead + "\nfix: " + e.Fix
}
func quote(s string) string {
	if runtime.GOOS == "windows" {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
func (r *Repository) fix(args string) string { return "git -C " + quote(r.Dir) + " " + args }
func (r *Repository) runAt(dir string, args ...string) (string, error) {
	// The data branch is not code: the host project's hooks (core.hooksPath is
	// shared by every worktree) check files that do not exist here and refused
	// huginn's commits in seid-lang, 2026-11-09. Never run them on this branch.
	if dir == r.Dir && len(args) > 0 && (args[0] == "commit" || args[0] == "rebase" || args[0] == "merge") {
		args = append([]string{"-c", "core.hooksPath=" + os.DevNull}, args...)
	}
	net := isNet(args)
	full := args
	if net {
		full = netArgs(args)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if net {
		ctx, cancel = context.WithTimeout(ctx, NetTimeout)
	}
	defer cancel()
	c := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, full...)...)
	// ssh may outlive a killed git and hold its output pipes: stop waiting.
	c.WaitDelay = 2 * time.Second
	c.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_EDITOR=true", "GIT_MERGE_AUTOEDIT=no")
	if net {
		c.Env = netEnv(dir, c.Env)
	}
	if r.Identity.Name != "" {
		c.Env = append(c.Env, "GIT_AUTHOR_NAME="+r.Identity.Name, "GIT_COMMITTER_NAME="+r.Identity.Name)
	}
	if r.Identity.Email != "" {
		c.Env = append(c.Env, "GIT_AUTHOR_EMAIL="+r.Identity.Email, "GIT_COMMITTER_EMAIL="+r.Identity.Email)
	}
	var stderr bytes.Buffer
	c.Stderr = &stderr
	b, err := c.Output()
	if err != nil {
		quoted := make([]string, len(args))
		for i, a := range args {
			quoted[i] = quote(a)
		}
		detail := strings.TrimSpace(string(b) + stderr.String() + " " + err.Error())
		fix := "git -C " + quote(dir) + " " + strings.Join(quoted, " ")
		if net && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return string(b), &Error{Op: strings.Join(args, " "), Detail: netTimeout(args), Fix: fix, Cause: ErrNetTimeout}
		}
		// An unreachable remote (offline, DNS, a moved path) is not a key
		// problem, though ssh ends both with "Could not read from remote
		// repository": say which (0hkaa5).
		unreachable := false
		for _, s := range []string{"Could not resolve hostname", "Could not resolve host", "connect to host", "Connection refused", "Network is unreachable", "Operation timed out", "Connection timed out", "does not appear to be a git repository", "Failed to connect to"} {
			unreachable = unreachable || strings.Contains(detail, s)
		}
		if net && unreachable {
			return string(b), &Error{Op: strings.Join(args, " "), Detail: "the remote is unreachable: " + detail, Fix: fix, Cause: err}
		}
		// A remote that refuses the key is not fixed by re-running the same
		// command. Apps (Claude, ChatGPT) start MCP servers without the
		// user's SSH agent, so say so (2026-11-16, Claude app report).
		if strings.Contains(detail, "Permission denied (publickey)") || strings.Contains(detail, "Could not read from remote repository") || strings.Contains(detail, "Host key verification failed") {
			detail = "the remote refused this process's SSH credentials (no SSH agent or key here; apps such as Claude Desktop start MCP servers without your agent): " + detail
			fix = "run this command in a terminal where 'ssh -T git@github.com' works, or give the app SSH_AUTH_SOCK / GIT_SSH_COMMAND; reading tasks (ls, show, next) works without the remote"
		}
		// A fresh machine or container without a git identity: re-running
		// the commit cannot help; setting the identity does (2026-11-16,
		// found running the linux release in a clean container).
		if strings.Contains(detail, "Author identity unknown") || strings.Contains(detail, "unable to auto-detect email address") {
			detail = "git has no author identity here, so task changes cannot be committed"
			fix = `git config --global user.name "Your Name" && git config --global user.email "you@example.com"`
		}
		return string(b), &Error{Op: strings.Join(args, " "), Detail: detail, Fix: fix, Cause: err}
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}
func (r *Repository) git(args ...string) (string, error) { return r.runAt(r.Dir, args...) }

// worktree is one entry of git worktree list --porcelain.
type worktree struct {
	Path, Branch   string // Branch is a full ref, empty when detached
	Bare, Prunable bool
}

func (r *Repository) worktrees(at string) ([]worktree, error) {
	out, err := r.runAt(at, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	var list []worktree
	for _, field := range strings.Split(out, "\x00") {
		key, value, _ := strings.Cut(field, " ")
		switch {
		case key == "worktree":
			list = append(list, worktree{Path: filepath.Clean(value)})
		case len(list) == 0:
		case key == "branch":
			list[len(list)-1].Branch = value
		case key == "bare":
			list[len(list)-1].Bare = true
		case key == "prunable":
			list[len(list)-1].Prunable = true
		}
	}
	return list, nil
}

// checkout returns the worktree that has branch checked out, if any.
func checkout(list []worktree, branch string) (worktree, bool) {
	for _, w := range list {
		if w.Branch == "refs/heads/"+branch {
			return w, true
		}
	}
	return worktree{}, false
}

// mainRoot is the main worktree (git lists it first). A bare repository has
// none, so the checkout itself is used.
func mainRoot(list []worktree, root string) string {
	if len(list) > 0 && !list[0].Bare && list[0].Path != "" {
		return list[0].Path
	}
	return root
}

// MainRoot returns the root of the main worktree of the repository that
// contains dir. From a linked worktree (git worktree add) that is the
// original checkout, not the linked one: branch checkouts made by Open live
// there, so every worktree shares them.
func MainRoot(dir string) (string, error) {
	l, err := Locate(dir)
	if err != nil {
		return "", err
	}
	return l.Main, nil
}

// Open opens or initializes the worktree. repoRoot may be any directory inside
// the host checkout, including a linked checkout. dir is relative to the main
// worktree's root: a branch has one checkout, shared by every linked worktree
// (065wwn), and a branch already checked out somewhere is used where it is.
// Callers do their file I/O under Repository.Dir.
func Open(repoRoot, branch, dir string, options ...Option) (*Repository, error) {
	return Init(repoRoot, branch, dir, options...)
}
func Init(repoRoot, branch, dir string, options ...Option) (*Repository, error) {
	loc, err := Locate(repoRoot)
	if err != nil {
		return nil, err
	}
	return loc.Open(branch, dir, options...)
}

// Location is what Open needs to know about the repository around a
// directory, found with two git processes (44w1ev: Open spawned eight, at
// ~15 ms each, before a command read a task). A caller that opens a branch
// and also asks about the repository (the main worktree, the common git
// dir) locates once and reuses it.
type Location struct {
	Root   string // top level of the worktree the directory is in
	Common string // absolute common git dir, shared by every worktree
	Main   string // top level of the main worktree (Root in a bare repo)
	list   []worktree
}

// Locate finds the Location of dir. Outside a git repository it fails.
func Locate(dir string) (*Location, error) {
	r := &Repository{Dir: dir}
	paths, err := r.runAt(dir, "rev-parse", "--path-format=absolute", "--show-toplevel", "--git-common-dir")
	if err != nil {
		// --path-format needs Git 2.31; say what to upgrade rather than
		// echoing an unknown-option error.
		if e := r.checkVersion(dir); e != nil {
			return nil, e
		}
		return nil, err
	}
	root, common, ok := strings.Cut(paths, "\n")
	if !ok {
		return nil, &Error{Op: "rev-parse", Detail: "unexpected output " + strconv.Quote(paths), Fix: "git -C " + quote(dir) + " rev-parse --path-format=absolute --show-toplevel --git-common-dir"}
	}
	l := &Location{Root: root, Common: strings.TrimSpace(common)}
	if l.list, err = r.worktrees(root); err != nil {
		return nil, err
	}
	l.Main = mainRoot(l.list, filepath.Clean(root))
	return l, nil
}

// Open opens or initializes branch's worktree at dir (relative to the main
// worktree) in the located repository; see the package-level Open.
func (l *Location) Open(branch, dir string, options ...Option) (*Repository, error) {
	r := &Repository{Dir: l.Root}
	for _, o := range options {
		o(r)
	}
	root, common, list, host := l.Root, l.Common, l.list, l.Main
	r.Root, r.Common = root, common
	var err error
	clean := filepath.Clean(dir)
	if filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return nil, &Error{Op: "init", Detail: "worktree directory must be inside checkout", Fix: "Choose a relative directory such as .notes"}
	}
	for _, part := range strings.Split(filepath.ToSlash(clean), "/") {
		if strings.EqualFold(part, ".git") || strings.ContainsAny(part, "\r\n") {
			return nil, &Error{Op: "init", Detail: "unsafe worktree directory", Fix: "git -C " + quote(root) + " worktree list"}
		}
	}
	// Reject symlink components rather than creating a worktree outside the host.
	ancestor := host
	for _, part := range strings.Split(filepath.ToSlash(clean), "/") {
		ancestor = filepath.Join(ancestor, part)
		info, e := os.Lstat(ancestor)
		if e != nil && !os.IsNotExist(e) {
			return nil, r.ioerr(e)
		}
		if e == nil && info.Mode()&os.ModeSymlink != 0 {
			return nil, &Error{Op: "init", Detail: "worktree directory traverses symlink: " + ancestor, Fix: "git -C " + quote(host) + " worktree list"}
		}
	}
	if _, ok := checkout(list, branch); !ok {
		// The branch is checked out somewhere only if git accepted its name.
		if _, err = r.runAt(root, "check-ref-format", "--branch", branch); err != nil {
			return nil, err
		}
	}
	r.Branch = branch
	r.Dir = filepath.Join(host, clean)
	if r.Remote, err = remoteOf(common); err != nil {
		if remotes, e := r.runAt(root, "remote"); e == nil {
			r.Remote = pickRemote(strings.Fields(remotes))
		} else {
			return nil, e
		}
	}
	exclude := filepath.Join(common, "info", "exclude")
	if err = os.MkdirAll(filepath.Dir(exclude), 0755); err != nil {
		return nil, r.ioerr(err)
	}
	b, err := os.ReadFile(exclude)
	if err != nil && !os.IsNotExist(err) {
		return nil, r.ioerr(err)
	}
	entry := "/" + strings.NewReplacer("\\", "\\\\", "*", "\\*", "?", "\\?", "[", "\\[", " ", "\\ ").Replace(filepath.ToSlash(clean)) + "/"
	found := false
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSuffix(line, "\r") == entry {
			found = true
		}
	}
	if !found {
		f, e := os.OpenFile(exclude, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if e != nil {
			return nil, r.ioerr(e)
		}
		prefix := ""
		if len(b) > 0 && b[len(b)-1] != '\n' {
			prefix = "\n"
		}
		_, e = f.WriteString(prefix + entry + "\n")
		ce := f.Close()
		if e != nil {
			return nil, r.ioerr(e)
		}
		if ce != nil {
			return nil, r.ioerr(ce)
		}
	}
	// One checkout per branch: a linked worktree uses the one that exists,
	// wherever it is (2026-11-16, 065wwn: a second worktree add failed with
	// "already used by worktree" and its fix repeated the same command).
	if w, ok := checkout(list, branch); ok {
		return r.reuse(w, host)
	}
	if _, e := os.Stat(filepath.Join(r.Dir, ".git")); e == nil {
		actual, e := r.git("symbolic-ref", "--short", "HEAD")
		if e != nil {
			// Git does not list this folder as a worktree (it was pruned or
			// copied): repeating the command cannot help. Keep the folder,
			// with any edits, and check the branch out afresh.
			stale := r.Dir + ".stale"
			return nil, &Error{Op: "open", Detail: r.Dir + " is not a working checkout of " + branch + " (git no longer knows it as a worktree)",
				Fix: "mv " + quote(r.Dir) + " " + quote(stale) + " && git -C " + quote(host) + " worktree prune && git -C " + quote(host) + " worktree add " + quote(r.Dir) + " " + quote(branch), Cause: e}
		}
		if actual != branch {
			return nil, &Error{Op: "open", Detail: "existing worktree uses branch " + actual, Fix: r.fix("switch " + quote(branch))}
		}
		return r, nil
	}
	if e := r.checkVersion(root); e != nil {
		return nil, e
	}
	exists := false
	if _, e := r.runAt(root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); e == nil {
		exists = true
	}
	remoteExists := false
	if r.Remote != "" {
		out, e := r.runAt(root, "ls-remote", "--heads", r.Remote, "refs/heads/"+branch)
		if e != nil {
			return nil, e
		}
		remoteExists = out != ""
		if remoteExists {
			if _, e = r.runAt(root, "fetch", r.Remote, "+refs/heads/"+branch+":refs/remotes/"+r.Remote+"/"+branch); e != nil {
				return nil, e
			}
		}
	}
	if exists {
		_, err = r.runAt(root, "worktree", "add", r.Dir, branch)
	} else if remoteExists {
		_, err = r.runAt(root, "worktree", "add", "-b", branch, r.Dir, "refs/remotes/"+r.Remote+"/"+branch)
	} else {
		_, err = r.runAt(root, "worktree", "add", "--orphan", "-b", branch, r.Dir)
	}
	if err != nil {
		// Another process may have checked the branch out meanwhile.
		if again, e := r.worktrees(root); e == nil {
			if w, ok := checkout(again, branch); ok {
				return r.reuse(w, host)
			}
		}
		return nil, err
	}
	return r, nil
}

// checkVersion fails unless git is at least 2.42 (worktree add --orphan).
func (r *Repository) checkVersion(dir string) error {
	version, err := r.runAt(dir, "--version")
	if err != nil {
		return err
	}
	var major, minor int
	fmt.Sscanf(version, "git version %d.%d", &major, &minor)
	if major < 2 || major == 2 && minor < 42 {
		fix := "sudo apt-get update && sudo apt-get install git && git --version"
		if runtime.GOOS == "darwin" {
			fix = "brew install git && git --version"
		}
		if runtime.GOOS == "windows" {
			fix = "winget upgrade --id Git.Git --exact"
		}
		return &Error{Op: "init", Detail: "Git >= 2.42 required for worktree add --orphan; found " + version, Fix: fix}
	}
	return nil
}

// pickRemote is origin when there is one, else the first remote, as
// git remote lists them.
func pickRemote(names []string) string {
	out := ""
	for _, v := range names {
		if out == "" || v == "origin" {
			out = v
		}
		if v == "origin" {
			break
		}
	}
	return out
}

// remoteOf answers git remote's choice from the repository's config file
// without spawning git: origin, when the file defines it, wins whatever else
// the global or system config adds. Anything else (no origin, an include,
// per-worktree config, no file) fails, and the caller asks git.
func remoteOf(common string) (string, error) {
	b, err := os.ReadFile(filepath.Join(common, "config"))
	if err != nil {
		return "", err
	}
	origin := false
	for _, line := range strings.Split(string(b), "\n") {
		low := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(low, "[include") || strings.HasPrefix(low, "worktreeconfig") {
			return "", fmt.Errorf("config reads other files")
		}
		if strings.TrimSpace(line) == `[remote "origin"]` {
			origin = true
		}
	}
	if !origin {
		return "", fmt.Errorf("no origin in %s", filepath.Join(common, "config"))
	}
	return "origin", nil
}

// reuse points r at the branch's existing checkout w.
func (r *Repository) reuse(w worktree, host string) (*Repository, error) {
	if w.Prunable {
		return nil, &Error{Op: "open", Detail: "branch " + r.Branch + " is registered as checked out at " + w.Path + " but that folder is gone",
			Fix: "git -C " + quote(host) + " worktree prune"}
	}
	r.Dir = w.Path
	return r, nil
}
func (r *Repository) ioerr(e error) error {
	return &Error{Op: "filesystem", Detail: e.Error(), Fix: r.fix("status"), Cause: e}
}

// Head returns an empty string for an unborn branch.
func (r *Repository) Head() (string, error) {
	h, e := r.git("rev-parse", "--verify", "HEAD")
	if e != nil {
		if _, x := r.git("symbolic-ref", "HEAD"); x == nil {
			if _, x = r.git("show-ref", "--verify", "--quiet", "refs/heads/"+r.Branch); exitCode(x) == 1 {
				return "", nil
			}
		}
	}
	return h, e
}

// Change uses Git status letters (A/M/D/R...), including rename similarity.
type Change struct{ Status, OldPath, Path string }

func (r *Repository) DiffNameStatus(old, new string) ([]Change, error) {
	if old == "" && new == "" {
		return nil, nil
	}
	newTree, e := r.git("rev-parse", "--verify", "--end-of-options", new+"^{tree}")
	if e != nil {
		return nil, e
	}
	if old == "" {
		out, e := r.git("ls-tree", "-r", "--name-only", "-z", newTree)
		if e != nil {
			return nil, e
		}
		var changes []Change
		for _, path := range strings.Split(strings.TrimSuffix(out, "\x00"), "\x00") {
			if path != "" {
				changes = append(changes, Change{Status: "A", Path: path})
			}
		}
		return changes, nil
	}
	oldTree, e := r.git("rev-parse", "--verify", "--end-of-options", old+"^{tree}")
	if e != nil {
		return nil, e
	}
	args := []string{"diff", "--name-status", "-z", "--find-renames", oldTree, newTree, "--"}
	out, e := r.git(args...)
	if e != nil {
		return nil, e
	}
	parts := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
	var result []Change
	for i := 0; i+1 < len(parts); {
		c := Change{Status: parts[i], Path: parts[i+1]}
		i += 2
		if strings.HasPrefix(c.Status, "R") || strings.HasPrefix(c.Status, "C") {
			if i >= len(parts) {
				return nil, &Error{Op: "diff", Detail: "invalid rename output", Fix: r.fix("status")}
			}
			c.OldPath = c.Path
			c.Path = parts[i]
			i++
		}
		result = append(result, c)
	}
	return result, nil
}

// Commit stages only paths in this worktree. Empty paths means all changes.
// Existing staged changes are refused so an explicit paths list cannot leak them.
func (r *Repository) Commit(paths []string, message string) error {
	if e := r.checkIdle(); e != nil {
		return e
	}
	staged, e := r.git("diff", "--cached", "--name-only")
	if e != nil {
		return e
	}
	if staged != "" {
		return &Error{Op: "commit", Detail: "index already contains staged changes", Fix: r.fix("restore --staged .")}
	}
	for _, p := range paths {
		c := filepath.Clean(p)
		if filepath.IsAbs(p) || c == ".." || strings.HasPrefix(c, ".."+string(filepath.Separator)) {
			return &Error{Op: "commit", Detail: "path escapes worktree: " + p, Fix: r.fix("status")}
		}
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	args := append([]string{"--literal-pathspecs", "add", "--all", "--"}, paths...)
	if _, e = r.git(args...); e != nil {
		return e
	}
	changed, e := r.git("diff", "--cached", "--name-only")
	if e != nil {
		return e
	}
	if changed == "" {
		return nil
	}
	_, e = r.git("commit", "-m", message)
	return e
}
func (r *Repository) remoteRef() string { return "refs/remotes/" + r.Remote + "/" + r.Branch }
func (r *Repository) fetch() (bool, error) {
	out, e := r.git("ls-remote", "--heads", r.Remote, "refs/heads/"+r.Branch)
	if e != nil {
		return false, e
	}
	if out == "" {
		return false, nil
	}
	_, e = r.git("fetch", r.Remote, "+refs/heads/"+r.Branch+":"+r.remoteRef())
	return true, e
}
func (r *Repository) pull() error {
	exists, e := r.fetch()
	if e != nil || !exists {
		return e
	}
	return r.integrate()
}

// integrate rebases local commits onto the fetched remote branch, without
// the network. A conflict aborts the rebase: local commits stay intact.
func (r *Repository) integrate() error {
	head, e := r.Head()
	if e != nil {
		return e
	}
	if head == "" {
		_, e = r.git("reset", "--hard", r.remoteRef())
		return e
	}
	if _, e = r.git("rebase", r.remoteRef()); e != nil {
		out, _ := r.git("diff", "--name-only", "--diff-filter=U", "-z")
		files := strings.Split(strings.TrimSuffix(out, "\x00"), "\x00")
		if _, abort := r.git("rebase", "--abort"); abort != nil {
			return abort
		}
		if out != "" {
			return &ConflictError{Files: files, Fix: r.fix("rebase " + r.remoteRef())}
		}
		return e
	}
	return nil
}
func (r *Repository) push() error {
	h, e := r.Head()
	if e != nil || h == "" {
		return e
	}
	_, e = r.git("push", r.Remote, "HEAD:refs/heads/"+r.Branch)
	return e
}

// Sync commits local edits then rebases and pushes only this branch.
func (r *Repository) Sync() error {
	if e := r.Commit(nil, "Sync "+r.Branch); e != nil {
		return e
	}
	if r.Remote == "" {
		return nil
	}
	if e := r.pull(); e != nil {
		return e
	}
	return r.push()
}

// Update implements push-as-lock. It requires a clean worktree so race recovery
// never discards unrelated edits. fn may read/write Dir but must not run Git.
// On callback failure its edits are preserved, without committing or pushing.
func (r *Repository) Update(fn func(*Repository) error) error {
	return r.UpdateMessage("Update "+r.Branch, fn)
}

// UpdateMessage is Update with the commit message to use.
func (r *Repository) UpdateMessage(message string, fn func(*Repository) error) error {
	if e := r.checkIdle(); e != nil {
		return e
	}
	status, e := r.git("status", "--porcelain")
	if e != nil {
		return e
	}
	if status != "" {
		return &Error{Op: "update", Detail: "worktree has uncommitted changes", Fix: r.fix("add --all") + " && " + r.fix("commit -m "+strconv.Quote("Save local changes"))}
	}
	if r.Remote != "" {
		if e = r.pull(); e != nil {
			return e
		}
	}
	if e = fn(r); e != nil {
		return e
	}
	if e = r.Commit(nil, message); e != nil {
		return e
	}
	if r.Remote == "" {
		return nil
	}
	if e = r.push(); e == nil {
		return nil
	}
	pushErr := e
	exists, e := r.fetch()
	if e != nil {
		return e
	}
	if !exists {
		return pushErr
	}
	// Distinguish a real non-fast-forward from authentication/hooks/network errors.
	if _, e = r.git("merge-base", "--is-ancestor", r.remoteRef(), "HEAD"); exitCode(e) != 1 {
		return pushErr
	}
	if _, e = r.git("reset", "--hard", r.remoteRef()); e != nil {
		return e
	}
	h, e := r.Head()
	if e != nil {
		return e
	}
	return &ErrRaced{RemoteHead: h, Dir: r.Dir, Fix: r.fix("show HEAD")}
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var e *exec.ExitError
	if errors.As(err, &e) {
		return e.ExitCode()
	}
	return -1
}

// Never adopt or abort an operation started by a human or another process.
func (r *Repository) checkIdle() error {
	dir, e := r.git("rev-parse", "--absolute-git-dir")
	if e != nil {
		return e
	}
	for _, item := range []struct{ path, abort string }{{"rebase-merge", "rebase --abort"}, {"rebase-apply", "rebase --abort"}, {"MERGE_HEAD", "merge --abort"}, {"CHERRY_PICK_HEAD", "cherry-pick --abort"}, {"REVERT_HEAD", "revert --abort"}} {
		if _, e := os.Stat(filepath.Join(dir, item.path)); e == nil {
			return &Error{Op: "worktree busy", Detail: "existing Git operation: " + item.path, Fix: r.fix(item.abort)}
		} else if !os.IsNotExist(e) {
			return r.ioerr(e)
		}
	}
	return nil
}

// Moved reports how Upgrade changed a checkout.
type Moved struct {
	FromBranch, ToBranch, FromDir, ToDir string
	LocalBranch, Worktree, RemoteBranch  bool // what was renamed or moved
}

// Upgrade moves a tool's branch and worktree to a new name and directory
// without changing history: git branch -m, then git worktree move. When a
// remote has the old branch and not the new one, the new name is pushed and
// the old remote branch is deleted (other checkouts upgrade themselves; their
// old local branch is found from the old remote-tracking ref). Running it
// again does nothing. It refuses while either worktree has uncommitted edits.
func Upgrade(repoRoot, fromBranch, fromDir, toBranch, toDir string, options ...Option) (Moved, error) {
	m := Moved{FromBranch: fromBranch, ToBranch: toBranch, FromDir: fromDir, ToDir: toDir}
	r := &Repository{Dir: repoRoot}
	for _, o := range options {
		o(r)
	}
	root, err := r.runAt(repoRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return m, err
	}
	// The checkouts belong to the main worktree, also when run from a linked one.
	if list, e := r.worktrees(root); e == nil {
		root = mainRoot(list, filepath.Clean(root))
	}
	r.Root, r.Dir = root, root
	has := func(ref string) bool { _, e := r.runAt(root, "show-ref", "--verify", "--quiet", ref); return e == nil }
	oldPath, newPath := filepath.Join(root, fromDir), filepath.Join(root, toDir)
	if _, e := os.Stat(filepath.Join(oldPath, ".git")); e == nil {
		if st, e := r.runAt(oldPath, "status", "--porcelain"); e != nil {
			return m, e
		} else if st != "" {
			return m, &Error{Op: "upgrade", Detail: fromDir + " has uncommitted changes", Fix: "git -C " + quote(oldPath) + " add --all && git -C " + quote(oldPath) + " commit -m " + strconv.Quote("Save before upgrade")}
		}
	}
	switch {
	case has("refs/heads/"+toBranch) && has("refs/heads/"+fromBranch):
		return m, &Error{Op: "upgrade", Detail: "both branches " + fromBranch + " and " + toBranch + " exist", Fix: "git -C " + quote(root) + " log --oneline -3 " + fromBranch + " " + toBranch}
	case has("refs/heads/" + fromBranch):
		if _, err = r.runAt(root, "branch", "-m", fromBranch, toBranch); err != nil {
			return m, err
		}
		m.LocalBranch = true
	}
	if _, e := os.Stat(filepath.Join(oldPath, ".git")); e == nil {
		if _, e := os.Stat(newPath); e == nil {
			return m, &Error{Op: "upgrade", Detail: toDir + " already exists", Fix: "ls -la " + quote(newPath)}
		}
		if err = os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
			return m, r.ioerr(err)
		}
		if _, err = r.runAt(root, "worktree", "move", oldPath, newPath); err != nil {
			return m, err
		}
		m.Worktree = true
	}
	remotes, err := r.runAt(root, "remote")
	if err != nil {
		return m, err
	}
	for _, rem := range strings.Fields(remotes) {
		if rem != "origin" && len(strings.Fields(remotes)) > 1 {
			continue
		}
		heads, e := r.runAt(root, "ls-remote", "--heads", rem, "refs/heads/"+fromBranch, "refs/heads/"+toBranch)
		if e != nil {
			return m, e
		}
		oldThere := strings.Contains(heads, "refs/heads/"+fromBranch)
		newThere := strings.Contains(heads, "refs/heads/"+toBranch)
		if oldThere && !newThere && has("refs/heads/"+toBranch) {
			if _, e = r.runAt(root, "push", rem, "refs/heads/"+toBranch+":refs/heads/"+toBranch); e != nil {
				return m, e
			}
			if _, e = r.runAt(root, "push", rem, "--delete", fromBranch); e != nil {
				return m, e
			}
			m.RemoteBranch = true
		}
		if has("refs/heads/" + toBranch) {
			r.runAt(root, "branch", "--set-upstream-to="+rem+"/"+toBranch, toBranch)
		}
		break
	}
	// The exclude entry for the old directory is harmless; add the new one.
	common, err := r.runAt(root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err == nil {
		exclude := filepath.Join(common, "info", "exclude")
		b, _ := os.ReadFile(exclude)
		entry := "/" + filepath.ToSlash(filepath.Clean(strings.SplitN(filepath.ToSlash(toDir), "/", 2)[0])) + "/"
		if !strings.Contains("\n"+string(b), "\n"+entry+"\n") {
			if f, e := os.OpenFile(exclude, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); e == nil {
				prefix := ""
				if len(b) > 0 && b[len(b)-1] != '\n' {
					prefix = "\n"
				}
				f.WriteString(prefix + entry + "\n")
				f.Close()
			}
		}
	}
	return m, nil
}

// EnsureFirstCommit makes an empty first commit on a branch with no commits
// yet, so the branch exists (and can be pushed) right after init.
func (r *Repository) EnsureFirstCommit(message string) error {
	h, e := r.Head()
	if e != nil || h != "" {
		return e
	}
	_, e = r.git("-c", "core.hooksPath="+os.DevNull, "commit", "--allow-empty", "-m", message)
	return e
}

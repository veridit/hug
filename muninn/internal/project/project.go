// Package project gives a working directory a stable identity that survives
// renames, moves and fresh clones, and syncs a project's memories with a
// JSONL file committed in the repository.
package project

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
)

// Project is a repository (or plain folder) muninn keeps memories for.
type Project struct {
	Client string // nearest client-root boundary, if any
	ID     string // stable: git root commit, or the id in .muninn/project
	Label  string // human name (the folder name)
	Root   string // top-level directory
	// Notes is the folder of the muninn-notes checkout. In git it is the main
	// worktree's .muninn-notes, also from a linked worktree (git worktree
	// add): memories follow the same one-checkout rule as tasks (065wwn).
	Notes string
}

// Scope is the recall scope of the project's own memories.
func (p Project) Scope() string {
	if p.Client != "" {
		return "client:" + p.Client + ":" + p.ID
	}
	return "project:" + p.ID
}

// File is the shared memory file inside the repository.
const NotesBranch = "muninn-notes"
const NotesDir = ".muninn-notes"

// File is the shared memory file: memories.jsonl in the notes checkout.
func (p Project) File() string {
	if p.Notes != "" {
		return filepath.Join(p.Notes, "memories.jsonl")
	}
	return filepath.Join(p.Root, NotesDir, "memories.jsonl")
}
func (p Project) LegacyFile() string { return filepath.Join(p.Root, ".muninn", "memories.jsonl") }

// Find resolves the project containing dir. In a git repository the id is
// the first root commit (identical for every clone, unchanged by renames
// and moves; the "rc-" prefix keeps it recognisable). Outside git, or in a
// repo without commits, the id lives in <root>/.muninn/project, created on
// first use when create is true.
func Find(dir string, create bool) (Project, error) {
	client, root, err := ClientRoot(dir)
	if err != nil {
		return Project{}, err
	}
	if client == "" {
		return find(dir, create)
	}
	p, err := find(dir, false)
	if err == nil {
		// root is physical (ClientRoot resolves symlinks); compare physical
		// paths too, or /tmp -> /private/tmp style links hide the project.
		pr := p.Root
		if phys, e := filepath.EvalSymlinks(pr); e == nil {
			pr = phys
		}
		rel, e := filepath.Rel(root, pr)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			err = ErrNoProject
		}
	}
	if err != nil {
		p, err = withFileID(root, create)
	}
	p.Client = client
	return p, err
}

func find(dir string, create bool) (Project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Project{}, err
	}
	key := findKey{abs, create}
	findMu.Lock()
	c, ok := findMemo[key]
	findMu.Unlock()
	if ok && time.Since(c.at) < findFor {
		return c.p, nil
	}
	p, err := findUncached(abs, create)
	if err == nil {
		findMu.Lock()
		findMemo[key] = foundAt{p, time.Now()}
		findMu.Unlock()
	}
	return p, err
}

// One command asks for the same folder's project several times (4vd1mz: a
// forget resolved its binding three times, two git processes each, ~150 ms
// of ~1.6 s). A found project is kept for findFor: long enough for a
// command, short enough that a long-lived server sees a new repository or
// its first commit. Failures are never kept.
const findFor = 2 * time.Second

type findKey struct {
	dir    string
	create bool
}

type foundAt struct {
	p  Project
	at time.Time
}

var (
	findMu   sync.Mutex
	findMemo = map[findKey]foundAt{}
)

// forgetFinds drops the memo (tests that change a repository in place).
func forgetFinds() {
	findMu.Lock()
	findMemo = map[findKey]foundAt{}
	findMu.Unlock()
}

func findUncached(abs string, create bool) (Project, error) {
	if top, err := gitTop(abs); err == nil && top != "" {
		top = filepath.Clean(top)
		p, err := gitProject(top)
		if errors.Is(err, errNoHistory) {
			p, err = withFileID(top, create)
		}
		if err == nil {
			// A main worktree has a .git folder and is its own main root: no
			// more git processes (ce3dmq: two each, ~45 ms on Windows, for
			// every placement lookup). A linked worktree (.git is a file)
			// asks git where its main checkout is.
			if st, e := os.Stat(filepath.Join(top, ".git")); e == nil && st.IsDir() {
				p.Notes = filepath.Join(top, NotesDir)
			} else if main, e := gitbranch.MainRoot(top); e == nil {
				p.Notes = filepath.Join(main, NotesDir)
			}
		}
		return p, err
	}
	// Not git: walk up to a folder that already has .muninn/project.
	for d := abs; ; d = filepath.Dir(d) {
		if id, err := readID(d); err == nil {
			return Project{ID: id, Label: filepath.Base(d), Root: d}, nil
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	return withFileID(abs, create)
}

// ErrNoProject means dir has no identity yet and create was false.
var ErrNoProject = errors.New("not a muninn project (no git history and no .muninn/project)")

// gitTop is the checkout's top level. Outside any repository (no .git in dir
// or a parent, no GIT_DIR) it starts no git process: plain folders paid a
// failing rev-parse per lookup, ~45 ms on Windows (ce3dmq).
func gitTop(dir string) (string, error) {
	if !inGit(dir) {
		return "", ErrNoProject
	}
	return git(dir, "rev-parse", "--show-toplevel")
}

func inGit(dir string) bool {
	if os.Getenv("GIT_DIR") != "" {
		return true
	}
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			return true
		}
		if filepath.Dir(d) == d {
			return false
		}
	}
}

// gitProject is the identity of the git checkout whose top level is top.
func gitProject(top string) (Project, error) {
	if id, err := readID(top); err == nil {
		return Project{ID: id, Label: filepath.Base(top), Root: top}, nil
	}
	if roots, err := git(top, "rev-list", "--max-parents=0", "HEAD"); err == nil && roots != "" {
		// Several roots (merged histories): the lexically smallest is
		// deterministic for every clone.
		ids := strings.Fields(roots)
		best := ids[0]
		for _, r := range ids[1:] {
			if r < best {
				best = r
			}
		}
		return Project{ID: "rc-" + best[:16], Label: filepath.Base(top), Root: top}, nil
	}
	return Project{}, errNoHistory
}

var errNoHistory = errors.New("git checkout without commits")

func withFileID(root string, create bool) (Project, error) {
	if !create {
		return Project{}, ErrNoProject
	}
	b := make([]byte, 8)
	rand.Read(b)
	id := "id-" + hex.EncodeToString(b)
	if err := os.MkdirAll(filepath.Join(root, ".muninn"), 0o755); err != nil {
		return Project{}, err
	}
	if err := os.WriteFile(filepath.Join(root, ".muninn", "project"), []byte(id+"\n"), 0o644); err != nil {
		return Project{}, err
	}
	return Project{ID: id, Label: filepath.Base(root), Root: root}, nil
}

// readID reads an explicit id; it wins over the git root commit so a team
// can pin an identity (e.g. after rewriting history).
func readID(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, ".muninn", "project"))
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(string(b))
	if id == "" {
		return "", errors.New("empty .muninn/project")
	}
	return id, nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

package board

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// The recall hook runs on every turn with a budget of tens of milliseconds,
// and each git subprocess costs ~15 ms. Where the repository layout is the
// common one (a .git folder, or a linked worktree's .git file, loose or
// packed refs), place and head are read from the files directly; anything
// else falls back to git.

// fastPlace finds the worktree, branch and main worktree of dir without git.
func fastPlace(dir string) (Place, string, bool) {
	top := dir
	for {
		fi, err := os.Lstat(filepath.Join(top, ".git"))
		if err == nil {
			p := Place{Worktree: top}
			gitdir := filepath.Join(top, ".git")
			if !fi.IsDir() {
				b, err := os.ReadFile(gitdir)
				if err != nil || !strings.HasPrefix(string(b), "gitdir: ") {
					return Place{}, "", false
				}
				gitdir = strings.TrimSpace(strings.TrimPrefix(string(b), "gitdir: "))
				if !filepath.IsAbs(gitdir) {
					gitdir = filepath.Join(top, gitdir)
				}
			}
			common := commonDir(gitdir)
			if filepath.Base(common) != ".git" {
				return Place{}, "", false // bare or unusual layout
			}
			p.Main = filepath.Dir(common)
			p.Linked = filepath.Clean(p.Main) != filepath.Clean(top)
			head, err := os.ReadFile(filepath.Join(gitdir, "HEAD"))
			if err != nil {
				return Place{}, "", false
			}
			if ref, ok := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: refs/heads/"); ok {
				p.Branch = ref
			}
			return p, common, true
		}
		parent := filepath.Dir(top)
		if parent == top {
			return Place{}, "", false
		}
		top = parent
	}
}

func commonDir(gitdir string) string {
	b, err := os.ReadFile(filepath.Join(gitdir, "commondir"))
	if err != nil {
		return filepath.Clean(gitdir)
	}
	c := strings.TrimSpace(string(b))
	if !filepath.IsAbs(c) {
		c = filepath.Join(gitdir, c)
	}
	return filepath.Clean(c)
}

// fastRef resolves refs/heads/<name> in the common dir: the loose file, then
// packed-refs. ok is false when neither has it (unborn, or a layout this
// does not read, such as reftable).
func fastRef(common, name string) (string, bool) {
	if b, err := os.ReadFile(filepath.Join(common, "refs", "heads", filepath.FromSlash(name))); err == nil {
		if h := strings.TrimSpace(string(b)); len(h) >= 40 && !strings.HasPrefix(h, "ref:") {
			return h, true
		}
		return "", false
	}
	f, err := os.Open(filepath.Join(common, "packed-refs"))
	if err != nil {
		return "", false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	want := " refs/heads/" + name
	for sc.Scan() {
		if l := sc.Text(); strings.HasSuffix(l, want) && len(l) > 40 {
			return l[:len(l)-len(want)], true
		}
	}
	return "", false
}

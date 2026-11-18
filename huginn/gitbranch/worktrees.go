package gitbranch

import (
	"os"
	"path/filepath"
	"strings"
)

// fastWorktrees is git worktree list from the files in the common git dir
// (0sbvnq: every hug command listed the worktrees, ~30 ms a process on
// Windows): the main worktree, then each <common>/worktrees/<id> with the
// folder its gitdir file names, its HEAD's branch, and prunable as git
// says it (the folder is gone and the entry is not locked). ok is false for
// anything it does not read with certainty (a bare repository, per-worktree
// config, a missing gitdir file); the caller asks git.
func fastWorktrees(common string) ([]worktree, bool) {
	if filepath.Base(common) != ".git" {
		return nil, false // bare, or a separate git dir
	}
	cfg, e := os.ReadFile(filepath.Join(common, "config"))
	if e != nil {
		return nil, false
	}
	for _, line := range strings.Split(string(cfg), "\n") {
		low := strings.ToLower(strings.Join(strings.Fields(line), ""))
		bare := low == "bare" || strings.HasPrefix(low, "bare=") && low != "bare=false"
		if bare || strings.HasPrefix(low, "worktree=") || strings.HasPrefix(low, "worktreeconfig") || strings.HasPrefix(low, "[include") {
			return nil, false
		}
	}
	main := worktree{Path: filepath.Clean(filepath.Dir(common))}
	if b, ok := headBranch(common); ok {
		main.Branch = b
	} else {
		return nil, false
	}
	list := []worktree{main}
	entries, e := os.ReadDir(filepath.Join(common, "worktrees"))
	if e != nil && !os.IsNotExist(e) {
		return nil, false
	}
	for _, d := range entries {
		if !d.IsDir() {
			return nil, false
		}
		admin := filepath.Join(common, "worktrees", d.Name())
		g, e := os.ReadFile(filepath.Join(admin, "gitdir"))
		if e != nil {
			return nil, false
		}
		dot := strings.TrimSpace(string(g))
		if dot == "" {
			return nil, false
		}
		if !filepath.IsAbs(dot) {
			dot = filepath.Join(admin, dot)
		}
		w := worktree{Path: filepath.Dir(filepath.Clean(dot))}
		b, ok := headBranch(admin)
		if !ok {
			return nil, false
		}
		w.Branch = b
		if _, e := os.Stat(dot); os.IsNotExist(e) {
			if _, e := os.Stat(filepath.Join(admin, "locked")); os.IsNotExist(e) {
				w.Prunable = true
			}
		} else if e != nil {
			return nil, false
		}
		list = append(list, w)
	}
	return list, true
}

// headBranch is the full ref HEAD in gitdir names ("" when detached).
func headBranch(gitdir string) (string, bool) {
	b, e := os.ReadFile(filepath.Join(gitdir, "HEAD"))
	if e != nil {
		return "", false
	}
	head := strings.TrimSpace(string(b))
	if ref, ok := strings.CutPrefix(head, "ref: "); ok {
		return ref, strings.HasPrefix(ref, "refs/heads/")
	}
	return "", isHash(head)
}

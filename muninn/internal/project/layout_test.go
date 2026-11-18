package project

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyMemoryLayoutRequiresExplicitUpgrade(t *testing.T) {
	for _, location := range []string{"checkout", "branch-only", "packed-branch", "remote-only", "linked-worktree"} {
		t.Run(location, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("JCODE_HOME", filepath.Join(home, "jcode"))
			t.Setenv("HUG_HOME", filepath.Join(home, "hug"))
			t.Setenv("HUG_SYNC", "off")
			root := t.TempDir()
			gitInit(t, root)
			branchGit(t, root, "branch", "muninn-notes")
			dir := root
			switch location {
			case "checkout", "linked-worktree":
				branchGit(t, root, "worktree", "add", filepath.Join(root, ".muninn-notes"), "muninn-notes")
				if location == "linked-worktree" {
					dir = filepath.Join(t.TempDir(), "linked")
					branchGit(t, root, "worktree", "add", "-b", "linked", dir)
				}
			case "packed-branch":
				branchGit(t, root, "pack-refs", "--all")
			case "remote-only":
				branchGit(t, root, "update-ref", "refs/remotes/origin/muninn-notes", "HEAD")
				branchGit(t, root, "branch", "-D", "muninn-notes")
			}
			refs := branchGit(t, root, "show-ref")
			for _, create := range []bool{false, true} {
				_, err := Find(dir, create)
				if err == nil || !strings.Contains(err.Error(), "fix: hug upgrade") {
					t.Fatalf("ordinary Find(create=%v) accepted legacy %s without explicit upgrade: %v", create, location, err)
				}
			}
			p := Project{Root: root, ID: "synthetic"}
			for _, dry := range []bool{false, true} {
				_, err := SyncBranch(p, filepath.Join(home, "native.json"), BranchOptions{DryRun: dry})
				if err == nil || !strings.Contains(err.Error(), "fix: hug upgrade") {
					t.Fatalf("ordinary SyncBranch(dry=%v) accepted legacy %s: %v", dry, location, err)
				}
			}
			if got := branchGit(t, root, "show-ref"); got != refs {
				t.Fatalf("ordinary use changed refs: %s", got)
			}
			if _, err := os.Stat(filepath.Join(root, ".hug", "memory")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("ordinary use created competing canonical checkout: %v", err)
			}
		})
	}
}

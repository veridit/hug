package gitbranch

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRefFilesAgreeWithGit: RefAt and fastHead read refs and HEAD from the
// files instead of spawning git (0sbvnq). Loose, packed, missing, a folder
// of the same name, a linked worktree, a detached HEAD and an unborn branch
// give git's answer, or ok=false so the caller asks git.
func TestRefFilesAgreeWithGit(t *testing.T) {
	root, _ := fixture(t, false)
	common := command(t, root, "rev-parse", "--path-format=absolute", "--git-common-dir")
	main := command(t, root, "rev-parse", "HEAD")
	check := func(ref, want string, wantFound bool) {
		t.Helper()
		h, found, ok := RefAt(common, ref)
		if !ok || found != wantFound || h != want {
			t.Fatalf("RefAt(%s) = %q found=%v ok=%v, git: %q found=%v", ref, h, found, ok, want, wantFound)
		}
	}
	check("refs/heads/main", main, true)
	check("refs/heads/huginn", "", false)
	// A folder named like the ref (refs/heads/huginn/x) is not the ref.
	command(t, root, "branch", "huginn/x")
	check("refs/heads/huginn", "", false)
	check("refs/heads/huginn/x", main, true)
	command(t, root, "update-ref", "refs/remotes/origin/huginn", "HEAD")
	command(t, root, "pack-refs", "--all")
	if _, e := os.Stat(filepath.Join(common, "refs", "remotes", "origin", "huginn")); e == nil {
		t.Fatal("pack-refs left the loose ref")
	}
	check("refs/remotes/origin/huginn", main, true)
	check("refs/heads/main", main, true)
	check("refs/remotes/origin/hug", "", false)

	// HEAD: on a branch (packed now), in a linked worktree, detached, unborn.
	if h, ok := fastHead(root); !ok || h != main {
		t.Fatalf("fastHead(main checkout) = %q %v, want %q", h, ok, main)
	}
	wt := filepath.Join(t.TempDir(), "wt")
	command(t, root, "worktree", "add", "-q", "-b", "side", wt)
	command(t, wt, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-q", "--allow-empty", "-m", "side")
	if h, ok := fastHead(wt); !ok || h != command(t, wt, "rev-parse", "HEAD") {
		t.Fatalf("fastHead(linked) = %q %v, git %q", h, ok, command(t, wt, "rev-parse", "HEAD"))
	}
	command(t, wt, "checkout", "-q", "--detach", "main")
	if h, ok := fastHead(wt); !ok || h != main {
		t.Fatalf("fastHead(detached) = %q %v, want %q", h, ok, main)
	}
	command(t, wt, "checkout", "-q", "--orphan", "fresh")
	if h, ok := fastHead(wt); ok {
		t.Fatalf("fastHead(unborn) = %q, want ok=false so git answers", h)
	}
	// Head keeps its answer for an unborn branch through the git path.
	r := openRepo(t, root, "hug/tasks", ".hug/tasks")
	if h, e := r.Head(); e != nil || h != "" {
		t.Fatalf("unborn Head = %q %v", h, e)
	}
}

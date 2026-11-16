package board

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/gitbranch"
)

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@e", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@e")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// The file-reading fast path agrees with git (gitbranch.Where, rev-parse) in
// the main checkout, a subfolder, and a linked worktree; refs resolve loose
// and packed.
func TestFastPathAgreesWithGit(t *testing.T) {
	repo, _ := filepath.EvalSymlinks(t.TempDir())
	run(t, repo, "init", "-q", "-b", "main")
	run(t, repo, "commit", "-q", "--allow-empty", "-m", "init")
	run(t, repo, "worktree", "add", "-q", "-b", "feat/x", filepath.Join(repo, "wt"))
	os.MkdirAll(filepath.Join(repo, "src", "deep"), 0o755)
	for _, dir := range []string{repo, filepath.Join(repo, "src", "deep"), filepath.Join(repo, "wt")} {
		fp, common, ok := fastPlace(dir)
		if !ok {
			t.Fatalf("%s: fast path refused", dir)
		}
		gp, err := gitbranch.Where(dir)
		if err != nil {
			t.Fatal(err)
		}
		if fp.Branch != gp.Branch || fp.Worktree != gp.Worktree || fp.Main != gp.Main || fp.Linked != gp.Linked {
			t.Fatalf("%s: fast %+v, git %+v", dir, fp, gp)
		}
		if want := run(t, dir, "rev-parse", "--path-format=absolute", "--git-common-dir"); common != want {
			t.Fatalf("%s: common %q, git %q", dir, common, want)
		}
	}
	_, common, _ := fastPlace(repo)
	want := run(t, repo, "rev-parse", "main")
	if h, ok := fastRef(common, "main"); !ok || h != want {
		t.Fatalf("loose ref: %q %v, want %q", h, ok, want)
	}
	run(t, repo, "pack-refs", "--all")
	if h, ok := fastRef(common, "feat/x"); !ok || h != want {
		t.Fatalf("packed ref: %q %v, want %q", h, ok, want)
	}
	if _, ok := fastRef(common, "hug/tasks"); ok {
		t.Fatal("missing ref resolved")
	}
}

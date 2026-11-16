package gitbranch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRemoteMatchesGit: Open reads the remote from the config file instead
// of spawning git remote (44w1ev). Whatever the config holds, the result is
// the one git remote gives: origin wins, else the first name git lists.
func TestRemoteMatchesGit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		remotes []string
	}{
		{"none", nil},
		{"origin", []string{"origin"}},
		{"origin among others", []string{"zeta", "origin", "alpha"}},
		{"no origin", []string{"zeta", "alpha"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := fixture(t, false)
			for _, n := range tc.remotes {
				bare := filepath.Join(t.TempDir(), n+".git")
				command(t, root, "init", "--bare", bare)
				command(t, root, "remote", "add", n, bare)
			}
			want := pickRemote(strings.Fields(command(t, root, "remote")))
			r := openRepo(t, root, "hug/tasks", ".hug/tasks")
			if r.Remote != want {
				t.Fatalf("Remote = %q, git remote gives %q", r.Remote, want)
			}
			common := command(t, root, "rev-parse", "--path-format=absolute", "--git-common-dir")
			if r.Common != common {
				t.Fatalf("Common = %q, want %q", r.Common, common)
			}
		})
	}
	// An include may define origin elsewhere: the file is not read, git is.
	root, _ := fixture(t, false)
	bare := filepath.Join(t.TempDir(), "inc.git")
	command(t, root, "init", "--bare", bare)
	extra := filepath.Join(t.TempDir(), "extra")
	must(t, os.WriteFile(extra, []byte("[remote \"origin\"]\n\turl = "+filepath.ToSlash(bare)+"\n"), 0644))
	command(t, root, "config", "include.path", extra)
	if r := openRepo(t, root, "hug/tasks", ".hug/tasks"); r.Remote != "origin" {
		t.Fatalf("included origin: Remote = %q", r.Remote)
	}
}

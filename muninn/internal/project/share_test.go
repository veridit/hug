package project

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/veridit/hug/muninn/internal/ingest"
)

// localDefault is the setting as the package ships it, before TestMain lets
// the older sync tests share (they exercise the push itself).
var localDefault func(string) (bool, error)

func TestMain(m *testing.M) {
	localDefault = shareSetting
	shareSetting = func(string) (bool, error) { return true, nil }
	os.Exit(m.Run())
}

func sharing(t *testing.T, f func(string) (bool, error)) {
	t.Helper()
	old := shareSetting
	shareSetting = f
	t.Cleanup(func() { shareSetting = old })
}

// 34r1ft check 2: a repository that says nothing keeps its memories local.
// Sync refuses: no hug/memory branch is made or pushed, nothing reaches the
// remote, and the memory is still in the local store.
func TestSyncRefusesToPushWhenRepoSaysLocal(t *testing.T) {
	sharing(t, localDefault)
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	os.MkdirAll(remote, 0o700)
	branchGit(t, remote, "init", "--bare", "--initial-branch=main")
	root := filepath.Join(base, "public")
	os.MkdirAll(root, 0o700)
	gitInit(t, root)
	branchGit(t, root, "remote", "add", "origin", remote)
	branchGit(t, root, "push", "origin", "HEAD:main")
	p, err := Find(root, false)
	if err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(base, "native.json")
	if _, _, err := ingest.Remember(native, "Synthetic project fact that must stay on this machine.", "fact", p.Scope(), nil); err != nil {
		t.Fatal(err)
	}
	for _, o := range []BranchOptions{{}, {DryRun: true}} {
		_, err = SyncBranch(p, native, o)
		if !errors.Is(err, ErrMemoriesLocal) {
			t.Fatalf("SyncBranch(%+v) on a repo that says nothing: err=%v, want ErrMemoriesLocal", o, err)
		}
	}
	if out, err := exec.Command("git", "-C", remote, "rev-parse", "--verify", "-q", "refs/heads/"+NotesBranch).CombinedOutput(); err == nil {
		t.Fatalf("remote got %s: %s", NotesBranch, out)
	}
	if out, err := exec.Command("git", "-C", root, "rev-parse", "--verify", "-q", "refs/heads/"+NotesBranch).CombinedOutput(); err == nil {
		t.Fatalf("local %s branch made: %s", NotesBranch, out)
	}
	if _, err := os.Stat(filepath.Join(root, NotesDir)); !os.IsNotExist(err) {
		t.Fatalf("%s checkout made: %v", NotesDir, err)
	}
	mems, err := ingest.LoadNative(native)
	if err != nil || len(mems) != 1 || !mems[0].Active {
		t.Fatalf("local memory lost: %+v %v", mems, err)
	}
}

// The default is local: the setting as shipped answers no for a repo with
// no setting at all (34r1ft principle: nothing is pushed unless the repo
// opts in).
func TestShareSettingDefaultsToLocal(t *testing.T) {
	root := t.TempDir()
	gitInit(t, root)
	share, err := localDefault(root)
	if err != nil || share {
		t.Fatalf("default setting: share=%v err=%v, want false", share, err)
	}
}

// A repository that keeps memories local starts no background pass: a
// pass could only refuse (34r1ft), so a remember there spawns nothing.
func TestNudgeSpawnsNothingWhenRepoSaysLocal(t *testing.T) {
	sharing(t, localDefault)
	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	os.MkdirAll(remote, 0o700)
	branchGit(t, remote, "init", "--bare", "--initial-branch=main")
	root := filepath.Join(base, "public")
	os.MkdirAll(root, 0o700)
	gitInit(t, root)
	branchGit(t, root, "remote", "add", "origin", remote)
	if nudgeWanted(root) {
		t.Fatalf("nudgeWanted(%s) = true on a repo that keeps memories local", root)
	}
	sharing(t, func(string) (bool, error) { return true, nil })
	if !nudgeWanted(root) {
		t.Fatalf("nudgeWanted(%s) = false on a repo that shares", root)
	}
}

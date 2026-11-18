package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/project"
)

// gitProject makes the test's folder a git repository with a remote, so
// remember --project stores in its shared project scope.
func gitProject(t *testing.T, dir string) project.Project {
	t.Helper()
	remote := filepath.Join(t.TempDir(), "remote.git")
	steps := [][]string{
		{"init", "-q", "--bare", "--initial-branch=main", remote},
		{"-C", dir, "init", "-q", "--initial-branch=main"},
		{"-C", dir, "-c", "user.name=Synthetic", "-c", "user.email=synthetic@example.invalid", "commit", "-q", "--allow-empty", "-m", "root"},
		{"-C", dir, "remote", "add", "origin", remote},
		{"-C", dir, "push", "-q", "origin", "HEAD:main"},
	}
	for _, args := range steps {
		cmd := exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git: %v %s", err, out)
		}
	}
	p, err := project.Find(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func storedSID(t *testing.T, text string) (string, bool) {
	t.Helper()
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range mems {
		if m.Content == text {
			return m.SID, m.Active
		}
	}
	t.Fatalf("%q is not in the store", text)
	return "", false
}

// tn8bmn check 5: remember prints the short id the store keeps, and a
// project sync never changes it. On 2026-11-18 remember printed yx7vxv, the
// background sync then gave the memory its repo origin, and the store and
// every clone called it erf4tk.
func TestRememberedSIDSurvivesProjectSync(t *testing.T) {
	home := memHome(t)
	p := gitProject(t, home)
	text := "Synthetic tn8bmn rule: the printed id is the kept id."
	out, errOut, code := runCLI(t, "", "remember", "--project", text)
	if code != 0 {
		t.Fatalf("remember: %d %s%s", code, out, errOut)
	}
	f := strings.Fields(out)
	if len(f) < 2 || f[0] != "remembered" {
		t.Fatalf("remember: %q", out)
	}
	printed := f[1]
	if sid, _ := storedSID(t, text); sid != printed {
		t.Fatalf("remember printed %s, the store holds %s", printed, sid)
	}
	if out, errOut, code = runCLI(t, "", "sync"); code != 0 {
		t.Fatalf("sync: %d %s%s", code, out, errOut)
	}
	if sid, _ := storedSID(t, text); sid != printed {
		t.Fatalf("remember printed %s, after sync the store holds %s", printed, sid)
	}
	// Another clone reads the notes file into its own store: the same id.
	other := filepath.Join(t.TempDir(), "other.json")
	if _, err := project.Sync(project.Project{ID: p.ID, Root: p.Root, Notes: filepath.Dir(p.File())}, other, false); err != nil {
		t.Fatal(err)
	}
	mems, err := ingest.LoadNative(other)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range mems {
		if m.Content == text {
			found = true
			if m.SID != printed {
				t.Fatalf("remember printed %s, the other clone holds %s", printed, m.SID)
			}
		}
	}
	if !found {
		t.Fatalf("the other clone did not get the memory")
	}
}

// tn8bmn check 3: forget reads its result back. When another writer puts
// the memory back before the readback, forget fails with a fix: line
// instead of printing "forgot".
func TestForgetUndoneByAnotherWriterFails(t *testing.T) {
	memHome(t)
	text := "Synthetic tn8bmn memory another writer restores."
	out, errOut, code := runCLI(t, "", "remember", text)
	if code != 0 {
		t.Fatalf("remember: %d %s%s", code, out, errOut)
	}
	sid := strings.Fields(out)[1]
	real := app.WriteLanded
	t.Cleanup(func() { app.WriteLanded = real })
	app.WriteLanded = func() {
		if err := ingest.UpdateNative(app.NativePath(), func(mems []ingest.NativeMemory) ([]ingest.NativeMemory, bool, error) {
			for i := range mems {
				if mems[i].SID == sid {
					mems[i].Active = true // a stale copy read back over the forget
				}
			}
			return mems, true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	out, errOut, code = runCLI(t, "", "forget", sid)
	if code == 0 || strings.Contains(out, "forgot") {
		t.Fatalf("forget reported success it does not have: %d %q %q", code, out, errOut)
	}
	if !strings.Contains(errOut, "forget did not hold: "+sid) || !strings.Contains(errOut, "fix: hug mem forget "+sid) {
		t.Fatalf("no readback error with a fix line: %q", errOut)
	}
	app.WriteLanded = real
	if out, errOut, code = runCLI(t, "", "forget", sid); code != 0 || !strings.Contains(out, "forgot "+sid) {
		t.Fatalf("the fix line's forget: %d %q %q", code, out, errOut)
	}
	if _, active := storedSID(t, text); active {
		t.Fatalf("%s is still active", sid)
	}
}

// tn8bmn check 3: remember reads its result back too: a memory another
// writer retired before the readback is no stored memory.
func TestRememberUndoneByAnotherWriterFails(t *testing.T) {
	memHome(t)
	text := "Synthetic tn8bmn memory another writer retires at once."
	real := app.WriteLanded
	t.Cleanup(func() { app.WriteLanded = real })
	app.WriteLanded = func() {
		if err := ingest.UpdateNative(app.NativePath(), func(mems []ingest.NativeMemory) ([]ingest.NativeMemory, bool, error) {
			for i := range mems {
				if mems[i].Content == text {
					mems[i].Active = false
				}
			}
			return mems, true, nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	out, errOut, code := runCLI(t, "", "remember", text)
	if code == 0 || strings.Contains(out, "remembered") {
		t.Fatalf("remember reported success it does not have: %d %q %q", code, out, errOut)
	}
	if !strings.Contains(errOut, "remember did not hold:") || !strings.Contains(errOut, "fix: hug mem show ") {
		t.Fatalf("no readback error with a fix line: %q", errOut)
	}
}

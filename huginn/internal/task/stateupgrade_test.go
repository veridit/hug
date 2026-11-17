package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameStateKey(t *testing.T) {
	for _, c := range []struct {
		name, in, want string
		did            bool
	}{
		{"renames", "---\nid: a\nstatus: \"doing\"\nstage: active\n---\n## Goal\nstatus: in the body stays\n", "---\nid: a\nstate: \"doing\"\nstage: active\n---\n## Goal\nstatus: in the body stays\n", true},
		{"crlf", "---\r\nid: a\r\nstatus: todo\r\n---\r\n", "---\r\nid: a\r\nstate: todo\r\n---\r\n", true},
		{"already", "---\nid: a\nstate: todo\n---\n", "---\nid: a\nstate: todo\n---\n", false},
		{"both keys are left for lint", "---\nstatus: todo\nstate: doing\n---\n", "---\nstatus: todo\nstate: doing\n---\n", false},
		{"no frontmatter", "status: todo\n", "status: todo\n", false},
		{"a field that starts with the word", "---\nstatus_note: x\nstate: todo\n---\n", "---\nstatus_note: x\nstate: todo\n---\n", false},
	} {
		got, did := RenameStateKey([]byte(c.in))
		if string(got) != c.want || did != c.did {
			t.Errorf("%s: got %q %v, want %q %v", c.name, got, did, c.want, c.did)
		}
		if HasLegacyStateKey([]byte(c.in)) != c.did {
			t.Errorf("%s: HasLegacyStateKey disagrees", c.name)
		}
		// Running it again changes nothing.
		if again, did := RenameStateKey(got); did || string(again) != string(got) {
			t.Errorf("%s: second run changed it: %q", c.name, again)
		}
	}
}

func TestUpgradeStateKeyBoard(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "One", Goal: "G", Acceptance: []string{"C"}, Type: "chore"})
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(s.Root, r.Task.Path)
	b, _ := os.ReadFile(path)
	// A board written by an older hug: force the old key.
	old := strings.Replace(string(b), "\nstate: ", "\nstatus: ", 1)
	if e := os.WriteFile(path, []byte(old), 0o644); e != nil {
		t.Fatal(e)
	}
	if !HasLegacyStateKey([]byte(old)) {
		t.Fatalf("fixture has no old key:\n%s", old)
	}
	changed, e := s.UpgradeStateKey()
	if e != nil || len(changed) != 1 || changed[0] != r.Task.Path {
		t.Fatalf("changed %v, %v", changed, e)
	}
	b, _ = os.ReadFile(path)
	if HasLegacyStateKey(b) || !strings.Contains(string(b), "\nstate: ") {
		t.Fatalf("not migrated:\n%s", b)
	}
	if changed, e := s.UpgradeStateKey(); e != nil || len(changed) != 0 {
		t.Fatalf("second run: %v %v", changed, e)
	}
}

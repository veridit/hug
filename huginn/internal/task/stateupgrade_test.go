package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpgradeFrontmatter(t *testing.T) {
	oslo := time.FixedZone("CET", 3600)
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
		{"zoneless times become UTC", "---\nstate: \"doing\"\ndoing_since: \"2026-11-17T14:30\"\nwaiting_since: 2026-11-17T08:05\r\n---\n- 2026-11-17T14:30 History stays\n",
			"---\nstate: \"doing\"\ndoing_since: \"2026-11-17T13:30:00Z\"\nwaiting_since: \"2026-11-17T07:05:00Z\"\r\n---\n- 2026-11-17T14:30 History stays\n", true},
		{"zoned times stay", "---\nstate: \"doing\"\ndoing_since: \"2026-11-17T13:30:00Z\"\ndone_at: \"2026-11-17T14:30:00+01:00\"\n---\n", "---\nstate: \"doing\"\ndoing_since: \"2026-11-17T13:30:00Z\"\ndone_at: \"2026-11-17T14:30:00+01:00\"\n---\n", false},
		{"not a time is left for lint", "---\nstate: \"doing\"\ndoing_since: \"yesterday\"\n---\n", "---\nstate: \"doing\"\ndoing_since: \"yesterday\"\n---\n", false},
	} {
		got, did := UpgradeFrontmatter([]byte(c.in), oslo)
		if string(got) != c.want || did != c.did {
			t.Errorf("%s: got %q %v, want %q %v", c.name, got, did, c.want, c.did)
		}
		if again, did := UpgradeFrontmatter(got, oslo); did || string(again) != string(got) {
			t.Errorf("%s: second run changed it: %q", c.name, again)
		}
	}
	if !HasLegacyStateKey([]byte("---\nstatus: todo\n---\n")) || HasLegacyStateKey([]byte("---\nstatus: todo\nstate: todo\n---\n")) || HasLegacyStateKey([]byte("---\nstate: todo\n---\nstatus: x\n")) {
		t.Fatal("HasLegacyStateKey")
	}
}

func TestStampIsUTC(t *testing.T) {
	old := Now
	Now = func() time.Time { return time.Date(2026, 11, 17, 14, 33, 0, 0, time.FixedZone("CET", 3600)) }
	t.Cleanup(func() { Now = old })
	if got := stamp(); got != "2026-11-17T13:33:00Z" {
		t.Fatalf("stamp %q", got)
	}
	for _, s := range []string{"2026-11-17T13:33:00Z", "2026-11-17T14:33:00+01:00"} {
		if at, ok := ParseStamp(s); !ok || !at.Equal(Now()) {
			t.Fatalf("parse %q: %v %v", s, at, ok)
		}
	}
	if _, ok := ParseStamp("2026-11-17T14:33"); !ok {
		t.Fatal("old form no longer read")
	}
}

// A zoneless time is a lint finding with fix: hug upgrade.
func TestLintZonelessTime(t *testing.T) {
	d := Parse("a.abc123.x.md", []byte("---\nid: abc123\ntitle: x\nstatus: doing\nstage: active\ndoing_since: \"2026-11-17T14:30\"\n---\n## Goal\ng\n## Acceptance\n- [ ] a\n"))
	n := 0
	for _, f := range d.Findings {
		if FindingCode(f.Problem) == "zoneless-time" {
			n++
			if f.Fix != "hug upgrade" || !strings.Contains(f.Problem, `doing_since is "2026-11-17T14:30"`) || f.Line != 6 {
				t.Fatalf("%+v", f)
			}
		}
	}
	if n != 1 {
		t.Fatalf("want one zoneless-time finding: %+v", d.Findings)
	}
}

func TestUpgradeFilesBoard(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "One", Goal: "G", Acceptance: []string{"C"}, Type: "chore"})
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(s.Root, r.Task.Path)
	b, _ := os.ReadFile(path)
	// A board written by an older hug: force the old key.
	old := strings.Replace(string(b), "\nstate: ", "\n"+legacyStateKey+": ", 1)
	if e := os.WriteFile(path, []byte(old), 0o644); e != nil {
		t.Fatal(e)
	}
	if !HasLegacyStateKey([]byte(old)) {
		t.Fatalf("fixture has no old key:\n%s", old)
	}
	changed, e := s.UpgradeFiles(time.UTC)
	if e != nil || len(changed) != 1 || changed[0] != r.Task.Path {
		t.Fatalf("changed %v, %v", changed, e)
	}
	b, _ = os.ReadFile(path)
	if HasLegacyStateKey(b) || !strings.Contains(string(b), "\nstate: ") {
		t.Fatalf("not migrated:\n%s", b)
	}
	if changed, e := s.UpgradeFiles(time.UTC); e != nil || len(changed) != 0 {
		t.Fatalf("second run: %v %v", changed, e)
	}
}

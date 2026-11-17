package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 6ephqk step 4: a file from before the rename says status: where state:
// goes. It still reads as its old state, lint names it old-layout with the
// fix hug upgrade, and MigrateState renames the key in place (value, quote
// and position kept), once.
func TestMigrateStateRenamesStatusKey(t *testing.T) {
	s, id := answerStore(t)
	tk, raw := readTask(t, s, id)
	if !strings.Contains(raw, "\nstate: todo\n") || strings.Contains(raw, "\nstatus:") {
		t.Fatalf("new task does not say state:\n%s", raw)
	}
	path := filepath.Join(s.Root, tk.Path)
	old := strings.Replace(raw, "\nstate: todo\n", "\nstatus: \"doing\"\n", 1)
	if e := os.WriteFile(path, []byte(old), 0o644); e != nil {
		t.Fatal(e)
	}
	s.Invalidate()

	d := Parse(tk.Path, []byte(old))
	if d.Task.State != "doing" {
		t.Fatalf("old layout reads as %q, want doing", d.Task.State)
	}
	var codes []string
	for _, f := range Lint([]*Document{d}) {
		codes = append(codes, FindingCode(f.Problem))
		if FindingCode(f.Problem) == "old-layout" && f.FixLine() != "hug upgrade" {
			t.Fatalf("old-layout fix %q", f.FixLine())
		}
		if strings.Contains(f.Problem, "missing field state") {
			t.Fatalf("old layout also reported as missing state: %v", f)
		}
	}
	if !strings.Contains(strings.Join(codes, " "), "old-layout") {
		t.Fatalf("no old-layout finding: %v", codes)
	}

	changed, lines, e := s.MigrateState(true)
	if e != nil || len(changed) != 1 || len(lines) != 1 || lines[0] != id+": status: doing -> state: doing" {
		t.Fatalf("dry run: %v %v %v", changed, lines, e)
	}
	if b, _ := os.ReadFile(path); string(b) != old {
		t.Fatalf("dry run wrote the file:\n%s", b)
	}
	if _, _, e = s.MigrateState(false); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(path)
	if want := strings.Replace(old, "\nstatus: \"doing\"\n", "\nstate: \"doing\"\n", 1); string(b) != want {
		t.Fatalf("migrated file:\n%s\nwant:\n%s", b, want)
	}
	if fs := Parse(tk.Path, b).Findings; len(fs) != 0 {
		t.Fatalf("migrated file has findings: %v", fs)
	}
	if changed, _, e = s.MigrateState(false); e != nil || len(changed) != 0 {
		t.Fatalf("second migrate: %v %v", changed, e)
	}
}

// Both keys: state: wins on read, and the migration drops the stray status:.
func TestMigrateStateDropsStrayStatus(t *testing.T) {
	raw := "---\nid: abc123\ntitle: \"T\"\nstate: done\nstatus: todo\ntype: chore\n---\n"
	d := Parse("abc123-t.md", []byte(raw))
	if d.Task.State != "done" {
		t.Fatalf("read %q, want done", d.Task.State)
	}
	did, e := d.RenameStatusKey()
	if e != nil || !did || strings.Contains(string(d.Bytes()), "status:") || !strings.Contains(string(d.Bytes()), "state: done") {
		t.Fatalf("did %v e %v:\n%s", did, e, d.Bytes())
	}
}

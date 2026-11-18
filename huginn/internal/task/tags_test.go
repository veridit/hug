package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// txjkrm check 2 (task half): hug.toml's top-level tags = [...] is the
// board's list; an open task's label not on it is an unlisted-tag
// finding (Advice). A variant of a listed tag (case, - or _, or a merge
// [tag_merges] states) has one fix, the replace naming the listed tag;
// any other has two: add it to the list, or remove it from the task.
// No list: nothing is judged. A closed task is history.
func TestTagsListAndFindings(t *testing.T) {
	root := t.TempDir()
	if got, e := Tags(root); e != nil || got.List != nil {
		t.Fatalf("no hug.toml: %+v %v", got, e)
	}
	ts := []Task{
		{ID: "aaaaaa", Path: "a/aaaaaa.md", State: "todo", Labels: []string{"compiler", "parser"}},
		{ID: "bbbbbb", Path: "b/bbbbbb.md", State: "doing", Labels: []string{"compiler", "one-off"}},
		{ID: "cccccc", Path: "c/cccccc.md", State: "done", Labels: []string{"one-off"}},
		{ID: "dddddd", Path: "d/dddddd.md", State: "todo", Labels: []string{"Compiler", "c_backend", "perf"}},
	}
	if fs := TagFindings(TagList{}, ts); len(fs) != 0 {
		t.Fatalf("no list judged: %v", fs)
	}
	toml := "layout = 3\ntags = [\"compiler\", \"parser\",\n  \"performance\", \"c-backend\"]\n\n[board]\ncoordinator = \"crab\"\ntags = [\"not-this\"]\n\n[tag_merges]\nperf = \"performance\"\n"
	if e := os.WriteFile(filepath.Join(root, ConfigName), []byte(toml), 0o644); e != nil {
		t.Fatal(e)
	}
	list, e := Tags(root)
	if e != nil || strings.Join(list.List, ",") != "compiler,parser,performance,c-backend" || list.Merges["perf"] != "performance" {
		t.Fatalf("tags: %+v %v", list, e)
	}
	fs := TagFindings(list, ts)
	got := map[string]Finding{}
	for _, f := range fs {
		got[f.Path+" "+strings.Fields(strings.TrimPrefix(f.Problem, "unlisted tag: "))[0]] = f
		if f.Code != "unlisted-tag" || FindingCode(f.Problem) != "unlisted-tag" {
			t.Fatalf("code: %+v", f)
		}
	}
	if len(fs) != 4 {
		t.Fatalf("findings: %+v", fs)
	}
	other := got["b/bbbbbb.md one-off"]
	if other.Problem != "unlisted tag: one-off is not in hug.toml tags" ||
		strings.Join(other.Fixes, " | ") != "hug task tags add one-off | hug task set bbbbbb --tag one-off=" || other.Fix != "hug task tags add one-off" {
		t.Fatalf("other: %+v", other)
	}
	for key, want := range map[string]string{
		"d/dddddd.md Compiler":  "hug task set dddddd --tag Compiler=compiler",
		"d/dddddd.md c_backend": "hug task set dddddd --tag c_backend=c-backend",
		"d/dddddd.md perf":      "hug task set dddddd --tag perf=performance",
	} {
		f := got[key]
		if f.Fix != want || len(f.Fixes) != 1 || !strings.Contains(f.Problem, "is a variant of "+strings.TrimPrefix(want[strings.Index(want, "=")+1:], "")) {
			t.Fatalf("%s: %+v", key, f)
		}
	}
	if Severity("unlisted-tag") != Advice {
		t.Fatalf("severity %q", Severity("unlisted-tag"))
	}
}

// WithTag adds name to the list, once, keeping every other line.
func TestWithTag(t *testing.T) {
	got := WithTag("layout = 3\ntags = [\"compiler\"]\n\n[board]\ncoordinator = \"crab\"\n", "perf")
	if got != "layout = 3\ntags = [\"compiler\", \"perf\"]\n\n[board]\ncoordinator = \"crab\"\n" {
		t.Fatalf("add:\n%s", got)
	}
	if again := WithTag(got, "perf"); again != got {
		t.Fatalf("twice:\n%s", again)
	}
	multi := WithTag("tags = [\"a\",\n  \"b\"]\nlayout = 3\n", "c")
	if multi != "tags = [\"a\", \"b\", \"c\"]\nlayout = 3\n" {
		t.Fatalf("multi-line:\n%s", multi)
	}
	if got := WithTag("layout = 3\n\n[board]\ncoordinator = \"crab\"\n", "parser"); got != "layout = 3\ntags = [\"parser\"]\n\n[board]\ncoordinator = \"crab\"\n" {
		t.Fatalf("first tag:\n%s", got)
	}
	if got := WithTag("[board]\ncoordinator = \"crab\"\n", "parser"); got != "tags = [\"parser\"]\n\n[board]\ncoordinator = \"crab\"\n" {
		t.Fatalf("table first:\n%s", got)
	}
	if got := WithTag("", "parser"); got != "tags = [\"parser\"]\n" {
		t.Fatalf("empty:\n%q", got)
	}
}

// hug task set ID --tag: NAME adds one, OLD=NEW replaces one, OLD=
// removes one; each run twice changes nothing the second time.
func TestSetTag(t *testing.T) {
	s, id := answerStore(t)
	d, _ := s.Get(id)
	if e := d.SetField("labels", []string{"Compiler", "perf"}); e != nil {
		t.Fatal(e)
	}
	if e := AtomicWrite(filepath.Join(s.Root, d.Task.Path), d.Bytes()); e != nil {
		t.Fatal(e)
	}
	labels := func() string { tk, _ := readTask(t, s, id); return strings.Join(tk.Labels, ",") }
	for _, step := range []struct{ spec, want string }{
		{"Compiler=compiler", "compiler,perf"},
		{"Compiler=compiler", "compiler,perf"},
		{"parser", "compiler,perf,parser"},
		{"parser", "compiler,perf,parser"},
		{"perf=", "compiler,parser"},
		{"perf=", "compiler,parser"},
		{"compiler=parser", "parser"},
	} {
		if _, e := s.Tag(id, step.spec); e != nil {
			t.Fatalf("%s: %v", step.spec, e)
		}
		if got := labels(); got != step.want {
			t.Fatalf("after --tag %s: %s, want %s", step.spec, got, step.want)
		}
	}
	if _, e := s.Tag(id, "=x"); e == nil || !strings.Contains(e.Error(), "fix: hug task set "+id+" --tag x") {
		t.Fatalf("empty OLD: %v", e)
	}
}

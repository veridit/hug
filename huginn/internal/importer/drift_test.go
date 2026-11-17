package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// 3mavpt: an update that only adds frontmatter fields the board copy lacks
// (Seid: parent: from parent_task_id, a mapping added after its import) is
// counted apart, with the fields; an update that changes content is not.
func TestFieldsOnlyUpdates(t *testing.T) {
	src, dst := setup(t)
	one := filepath.Join(src, "one.md")
	b, _ := os.ReadFile(one)
	write(t, one, []byte(strings.Replace(string(b), "ordinal: 1\n", "ordinal: 1\nparent_task_id: DEMO-002\n", 1)))
	run(t, src, dst, Options{})
	// Play an older converter (Seid's import, before parent_task_id
	// became parent:): no parent: in the board copy or its baseline.
	d, e := dst.Get("DEMO-001")
	if e != nil {
		t.Fatal(e)
	}
	strip := func(s string) string {
		var out []string
		for _, l := range strings.SplitAfter(s, "\n") {
			if !strings.HasPrefix(l, "parent: ") {
				out = append(out, l)
			}
		}
		return strings.Join(out, "")
	}
	board := filepath.Join(dst.Root, d.Task.Path)
	local, _ := os.ReadFile(board)
	if !strings.Contains(string(local), "parent: ") {
		t.Fatalf("fixture: no parent: in\n%s", local)
	}
	write(t, board, []byte(strip(string(local))))
	base := filepath.Join(dst.Root, ".backlog-import", d.Task.ID+".json")
	var raw []byte
	bj, _ := os.ReadFile(base)
	if e := json.Unmarshal(bj, &raw); e != nil {
		t.Fatal(e)
	}
	bj, _ = json.Marshal([]byte(strip(string(raw))))
	write(t, base, bj)
	r := run(t, src, dst, Options{DryRun: true})
	if r.Updated != 1 || r.FieldsOnly != 1 || r.Fields["parent"] != 1 || len(r.Fields) != 1 {
		t.Fatalf("parent only: updated %d fields-only %d fields %v", r.Updated, r.FieldsOnly, r.Fields)
	}
	// A content change is not fields-only.
	b, _ = os.ReadFile(one)
	write(t, one, []byte(strings.Replace(string(b), "Synthetic goal.", "Upstream goal.", 1)))
	r = run(t, src, dst, Options{DryRun: true})
	if r.Updated != 1 || r.FieldsOnly != 0 {
		t.Fatalf("content change: updated %d fields-only %d", r.Updated, r.FieldsOnly)
	}
}

func TestOnlyAddsFields(t *testing.T) {
	local := "---\nid: \"a\"\ntitle: \"T\"\n---\n\n## Goal\nG\n"
	cases := []struct {
		result string
		want   bool
		fields string
	}{
		{"---\nid: \"a\"\ntitle: \"T\"\nparent: \"p\"\n---\n\n## Goal\nG\n", true, "parent"},
		{"---\nid: \"a\"\nparent: \"p\"\ntitle: \"T\"\n---\n\n## Goal\nG\n", true, "parent"},
		{"---\nid: \"a\"\ntitle: \"U\"\nparent: \"p\"\n---\n\n## Goal\nG\n", false, ""},
		{"---\nid: \"a\"\ntitle: \"T\"\n---\n\n## Goal\nG2\n", false, ""},
		{"---\nid: \"a\"\ntitle: \"T\"\n---\n\n## Goal\nG\n\n## New\nx\n", false, ""},
		{"---\nid: \"a\"\n---\n\n## Goal\nG\n", false, ""},
	}
	for _, c := range cases {
		f, ok := onlyAddsFields([]byte(local), []byte(c.result))
		if ok != c.want || strings.Join(f, ",") != c.fields {
			t.Errorf("%q: got %v %v", c.result, f, ok)
		}
	}
}

// converterPin is the converter's output on the test fixture under
// ConverterVersion. When this test fails, the converter's output changed:
// bump ConverterVersion and record the new hash here. hug setup keys its
// remembered dry run on ConverterVersion (3mavpt), so an output change
// without a bump would leave setup reporting an old answer.
var converterPin = map[int]string{
	1: "8ac2f3680459b407d111fd01906709d20fee3be13ff7ceb248dfe0e0222f6987",
}

func TestConverterVersionPinned(t *testing.T) {
	src, dst := setup(t)
	write(t, filepath.Join(src, "three.md"), []byte(strings.NewReplacer("DEMO-001", "DEMO-003", "ordinal: 1\n", "ordinal: 3\nparent_task_id: DEMO-001\ntype: enhancement\ncreated_date: '2026-01-02 10:00'\nupdated_date: '2026-01-03 11:00'\n", "status: To Do", "status: Waiting").Replace(fixture)))
	run(t, src, dst, Options{})
	h := sha256.New()
	ts, e := dst.Tasks()
	if e != nil {
		t.Fatal(e)
	}
	for _, x := range ts {
		b, e := os.ReadFile(filepath.Join(dst.Root, x.Path))
		if e != nil {
			t.Fatal(e)
		}
		// Ids are derived from the alias, so stable; paths too.
		h.Write([]byte(x.Path + "\n"))
		h.Write(b)
	}
	got := hex.EncodeToString(h.Sum(nil))
	want, ok := converterPin[ConverterVersion]
	if !ok || want != got {
		t.Fatalf("converter output changed under ConverterVersion %d: got %s, pinned %q.\nfix: bump importer.ConverterVersion and pin %d: %q", ConverterVersion, got, want, ConverterVersion+boolInt(ok && want != ""), got)
	}
	_ = task.Layout
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

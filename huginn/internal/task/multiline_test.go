package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// t3fqaw: Seid's imported titles are YAML block scalars over several lines
// (e6ajk8 before 2e018cb9a: "title: >-" then two indented lines). Setting a
// field replaces the whole value, continuation lines included.
// The title of Seid's t6j13s before dc0e38964 (2026-11-16 21:22, hug
// v0.7.35): its third line starts with "#", which parsing took for a YAML
// comment, so set --title left "  #return/#break clauses ..." behind.
const foldedTitle = "---\nid: \"e6ajk8\"\ntitle: >-\n  [Bug][Typecheck] handler return clause: 'This expression returns Option Int\n  but this function promises Option Int' (same type both sides) on a grant with\n  #return/#break clauses (test_handler_return.sd)\nstate: \"todo\"\ntype: \"bug\"\npriority: \"medium\"\nlabels: []\nassignee: []\ncreated: \"2026-10-19\"\nupdated: \"2026-10-19\"\n---\n\n## Goal\ng\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [ ] a\n"

func multiLineStore(t *testing.T, body string) *Store {
	t.Helper()
	s := &Store{Root: t.TempDir()}
	if e := os.WriteFile(filepath.Join(s.Root, "vf.e6ajk8.bug.md"), []byte(body), 0o644); e != nil {
		t.Fatal(e)
	}
	return s
}

// frontLines is the front matter between the --- lines.
func frontLines(t *testing.T, s *Store) []string {
	t.Helper()
	_, text := load(t, s, "e6ajk8")
	parts := strings.SplitN(text, "---\n", 3)
	if len(parts) < 3 {
		t.Fatalf("no front matter:\n%s", text)
	}
	return strings.Split(strings.TrimSuffix(parts[1], "\n"), "\n")
}

func lintClean(t *testing.T, s *Store) {
	t.Helper()
	d, e := s.Get("e6ajk8")
	if e != nil {
		t.Fatal(e)
	}
	for _, f := range Lint([]*Document{d}) {
		if f.Severity() == "broken" || strings.Contains(f.Problem, "frontmatter") || strings.Contains(f.Problem, "missing") || strings.Contains(f.Problem, "unknown") {
			t.Errorf("lint after the write: %s (line %d)", f.Problem, f.Line)
		}
	}
}

// Check 1: set --title on a 3-line folded title leaves one title line.
func TestSetTitleReplacesFoldedTitle(t *testing.T) {
	t.Parallel()
	s := multiLineStore(t, foldedTitle)
	if d, e := s.Get("e6ajk8"); e != nil || !strings.HasSuffix(d.Task.Title, "#return/#break clauses (test_handler_return.sd)") {
		t.Fatalf("fixture does not parse: %v", e)
	}
	if _, e := s.Set("e6ajk8", "title", "Handler grant #return/#break refuses conversion between identical Option Int"); e != nil {
		t.Fatal(e)
	}
	front := frontLines(t, s)
	for _, l := range front {
		if strings.HasPrefix(l, " ") {
			t.Errorf("continuation line left: %q\n%s", l, strings.Join(front, "\n"))
		}
	}
	if front[1] != `title: "Handler grant #return/#break refuses conversion between identical Option Int"` {
		t.Errorf("title line is %q", front[1])
	}
	if d, _ := s.Get("e6ajk8"); d.Task.State != "todo" || d.Task.Type != "bug" {
		t.Errorf("fields after the title: state %q type %q", d.Task.State, d.Task.Type)
	}
	lintClean(t, s)
}

// Check 2: every front-matter field a command writes replaces a multi-line
// value whole: each field in turn is a block scalar, then written.
func TestEveryFieldWriteReplacesMultiLineValue(t *testing.T) {
	t.Parallel()
	// Every old value line starts with "#" after its indent, the shape
	// that was taken for a comment.
	block := func(field string) string {
		return field + ": >-\n  first line of an old value\n  #second line\n  #third line\n"
	}
	cases := []struct {
		field string
		write func(s *Store) error
	}{
		{"title", func(s *Store) error { _, e := s.Set("e6ajk8", "title", "Short"); return e }},
		{"type", func(s *Store) error { _, e := s.Set("e6ajk8", "type", "feature"); return e }},
		{"priority", func(s *Store) error { _, e := s.Set("e6ajk8", "priority", "high"); return e }},
		{"assignee", func(s *Store) error { _, e := s.Set("e6ajk8", "assignee", "kestrel"); return e }},
		{"labels", func(s *Store) error { _, e := s.Set("e6ajk8", "label", "x"); return e }},
		{"waiting_on", func(s *Store) error {
			if _, e := s.Take("e6ajk8", "kestrel"); e != nil {
				return e
			}
			_, e := s.Wait("e6ajk8", "CI run 2", "kestrel")
			return e
		}},
		{"design_question", func(s *Store) error { _, e := s.Design("e6ajk8", "which way?", "kestrel", "jhf"); return e }},
	}
	for _, c := range cases {
		t.Run(c.field, func(t *testing.T) {
			t.Parallel()
			// The field's own line becomes a 3-line block scalar; a field
			// the file does not have yet goes in before state, with an
			// old value its writer must replace.
			lines := strings.Split(foldedTitle, "\n")
			found := false
			for i, l := range lines {
				if strings.HasPrefix(l, c.field+": ") {
					lines[i], found = strings.TrimSuffix(block(c.field), "\n"), true
				}
			}
			if !found {
				for i, l := range lines {
					if strings.HasPrefix(l, "state: ") {
						lines[i] = strings.TrimSuffix(block(c.field), "\n") + "\n" + l
						break
					}
				}
			}
			s := multiLineStore(t, strings.Join(lines, "\n"))
			if e := c.write(s); e != nil {
				t.Fatal(e)
			}
			for _, l := range frontLines(t, s) {
				if strings.Contains(l, "line of an old value") || strings.TrimSpace(l) == "#second line" || strings.TrimSpace(l) == "#third line" {
					t.Errorf("%s: old value line left: %q", c.field, l)
				}
			}
			lintClean(t, s)
		})
	}
}

// A comment at the key's column after a value is not the value: setting the
// field keeps it.
func TestSetKeepsCommentAfterValue(t *testing.T) {
	t.Parallel()
	body := strings.Replace(foldedTitle, "state: \"todo\"\n", "# kept: a note to the reader\nstate: \"todo\"\n", 1)
	s := multiLineStore(t, body)
	if _, e := s.Set("e6ajk8", "title", "Short"); e != nil {
		t.Fatal(e)
	}
	front := strings.Join(frontLines(t, s), "\n")
	if !strings.Contains(front, "title: \"Short\"\n# kept: a note to the reader\nstate: \"todo\"") {
		t.Fatalf("the comment moved or went:\n%s", front)
	}
}

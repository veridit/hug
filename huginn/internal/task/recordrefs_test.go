package task

import (
	"strings"
	"testing"
)

// missingFiles are the missing-file findings lint reports on s.
func missingFiles(t *testing.T, s *Store) []string {
	t.Helper()
	fs, err := s.LintFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range fs {
		if FindingCode(f.Problem) == "missing-file" {
			out = append(out, f.Path+": "+f.Problem)
		}
	}
	return out
}

// 7j706s check 1 (p87ngv question 5, answered): a path quoted in an
// answered question, and again in the ## History block the answer wrote,
// is record of the past, not a link: no missing-file. A dangling file:
// link in a live section (Notes) still is.
func TestRecordQuotesAreNoLinks(t *testing.T) {
	s, id := answerStore(t)
	tk, _ := readTask(t, s, id)
	folder := FolderOf(tk)
	if _, e := s.TakeAs(id, "crab", "claude-opus-5-5"); e != nil {
		t.Fatal(e)
	}
	q := "History file: one append-only history.md per task in its files (.hug/tasks/" + folder + "/history.md), see file:history.md. Is that the right name?"
	if _, e := s.Ask(id, Question{What: q, For: "jhf"}, "crab"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Answer(id, "history.sdd, Seid data", "jhf"); e != nil {
		t.Fatal(e)
	}
	_, raw := readTask(t, s, id)
	hist := raw[strings.Index(raw, "## History"):]
	if !strings.Contains(hist, ".hug/tasks/"+folder+"/history.md") || !strings.Contains(raw, "  answered ") {
		t.Fatalf("setup: the path is not quoted in the answered question and History:\n%s", raw)
	}
	if got := missingFiles(t, s); len(got) != 0 {
		t.Fatalf("a path quoted in an answered question and in History: want no missing-file, got %q", got)
	}
	// A dangling file: link in a live section is still broken.
	if _, e := s.Note(id, "evidence in file:gone.tsv"); e != nil {
		t.Fatal(e)
	}
	if got := missingFiles(t, s); len(got) != 1 || !strings.Contains(got[0], "file:gone.tsv: file not found") {
		t.Fatalf("a dangling file: link in Notes: want one missing-file for gone.tsv, got %q", got)
	}
}

// 7j706s check 3 (filing 7j706s raised missing-file on itself): Purpose
// prose that quotes a task path, of this task or another on the board, is
// no link; only file: links (and task:REF/NAME) are checked.
func TestProseQuotedPathIsNoLink(t *testing.T) {
	s := &Store{Root: t.TempDir(), DepsMode: "warn"}
	other, err := s.New(NewOptions{Title: "Marks", Goal: "g", Acceptance: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	of := FolderOf(other.Task)
	r, err := s.New(NewOptions{Title: "Lint", Goal: "g", Acceptance: []string{"a"},
		Purpose: "lint reports .hug/tasks/" + of + "/history.md, a name crab proposed; also `.hug/tasks/" + of + "/plan.md`",
		Details: "and \".hug/tasks/" + of + "/notes.txt\" in the details"})
	if err != nil {
		t.Fatal(err)
	}
	if got := missingFiles(t, s); len(got) != 0 {
		t.Fatalf("paths quoted in Purpose and Relevant details: want no missing-file, got %q", got)
	}
	// An explicit link in the same prose is still checked.
	if _, e := s.Note(r.Task.ID, "the plan is task:"+other.Task.ID+"/plan.md"); e != nil {
		t.Fatal(e)
	}
	if got := missingFiles(t, s); len(got) != 1 || !strings.Contains(got[0], "task:"+other.Task.ID+"/plan.md: file not found") {
		t.Fatalf("a dangling task: link: want one missing-file, got %q", got)
	}
}

// 7j706s: a resolved request is record too: its text and the account of
// what was done quote the past. An open request is live and still checked.
func TestResolvedRequestIsRecord(t *testing.T) {
	s, id := answerStore(t)
	if _, e := s.Request(id, "please attach file:shot.png", "jhf"); e != nil {
		t.Fatal(e)
	}
	if got := missingFiles(t, s); len(got) != 1 || !strings.Contains(got[0], "file:shot.png: file not found") {
		t.Fatalf("an open request citing a missing file: want one missing-file, got %q", got)
	}
	if _, e := s.Resolve(id, 1, "not needed, see file:old.png", "crab"); e != nil {
		t.Fatal(e)
	}
	if got := missingFiles(t, s); len(got) != 0 {
		t.Fatalf("a resolved request: want no missing-file, got %q", got)
	}
}

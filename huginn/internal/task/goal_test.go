package task

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// 0pr03h: the sentence rule. Conservative: when unsure, one sentence.
func TestGoalSentences(t *testing.T) {
	for _, c := range []struct {
		text string
		want int
	}{
		// One sentence: none of these split.
		{"Submit works.", 1},
		{"Ship it, e.g. as v0.7.85 in main.go.", 1},
		{"Use a flag, e.g. --goal, i.e. one write, then done", 1},
		{"Compare vs. Opus and cf. task:abc123 for the rest", 1},
		{"Read huginn/internal/task/goal.go and README.md before v0.7.85.", 1},
		{"The U.S. Bank feed works", 1},
		{"Written by J. Smith in 2026", 1},
		{"Steps: 1. Do it", 1},
		{"Is it fast? yes, under 2 s", 1},
		{"Wait... then go", 1},
		{"Keep 3.5 MB and 1.2.3 apart", 1},
		{"Run `hug task lint`. it warns", 1},
		{"Hello world", 1},
		// Two or more.
		{"It works. Then it ships.", 2},
		{"Is it fast? Yes.", 2},
		{"Done! Next.", 2},
		{"It works (fast). Then it ships", 2},
		{`He said "stop." Then left`, 2},
		{"One line\n\nAnother paragraph", 2},
		{"First. Second. Third.", 3},
		{"It ships. \"Quoted\" next", 2},
	} {
		if got := Sentences(c.text); got != c.want {
			t.Errorf("Sentences(%q) = %d, want %d", c.text, got, c.want)
		}
	}
}

func TestGoalProblem(t *testing.T) {
	if _, bad := GoalProblem(strings.Repeat("a", MaxGoal)); bad {
		t.Error("140 characters is allowed")
	}
	if _, bad := GoalProblem(""); bad {
		t.Error("empty is missing-goal's")
	}
	p, bad := GoalProblem(strings.Repeat("é", MaxGoal+1))
	if !bad || p.Code != "goal-too-long" || !strings.HasPrefix(p.Text, "Goal is 141 characters: ") || p.Flag != `--goal "one sentence" --details "..."` {
		t.Fatalf("%v %+v", bad, p)
	}
	p, _ = GoalProblem("It works. Then it ships.")
	if !strings.HasPrefix(p.Text, "Goal is 2 sentences: the Goal is one sentence of at most 140 characters; detail goes in Details") || FindingCode(p.Text) != "goal-too-long" {
		t.Fatal(p.Text)
	}
	p, _ = GoalProblem(strings.Repeat("Word. ", 30))
	if !strings.HasPrefix(p.Text, "Goal is 179 characters in 30 sentences: ") {
		t.Fatal(p.Text)
	}
}

// 0pr03h check 1: new refuses a long Goal with every other problem and one
// fix: line; the TUI add and promote (no Pattern) keep any Goal.
func TestNewRefusesLongGoal(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	_, e := s.New(NewOptions{Title: "T", Goal: "It works. Then it ships.", Purpose: "P", Principle: "Q", Details: "D", Acceptance: []string{"C"}, Pattern: true})
	var te *Error
	if !errors.As(e, &te) || te.Kind != "incomplete-task" || len(te.Problems) != 1 || te.Problems[0].Code != "goal-too-long" {
		t.Fatal(e)
	}
	if want := `hug task new "T" --goal "one sentence" --purpose "P" --principle "Q" --details "..." --accept "C"`; te.Fix != want {
		t.Fatalf("fix %q, want %q", te.Fix, want)
	}
	if !strings.HasSuffix(te.Error(), "fix: "+te.Fix) && !strings.Contains(te.Error(), te.Fix) {
		t.Fatal(te.Error())
	}
	if ts, _ := s.Tasks(); len(ts) != 0 {
		t.Fatal("refused new wrote a task")
	}
	if _, e := s.New(NewOptions{Title: "T", Goal: "Ship it, e.g. as v0.7.85 in main.go.", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}, Pattern: true}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.New(NewOptions{Title: "Quick", Goal: "Two. Sentences here.", Acceptance: []string{"C"}}); e != nil {
		t.Fatalf("TUI add refused: %v", e)
	}
}

// 0pr03h check 1: set refuses a long Goal, writes nothing, and the fix
// runs; a long Goal on disk is read as before and other fields still set.
func TestSetRefusesLongGoalAndKeepsOld(t *testing.T) {
	s := &Store{Root: t.TempDir()}
	r, e := s.New(NewOptions{Title: "T", Goal: "g", Acceptance: []string{"C"}})
	if e != nil {
		t.Fatal(e)
	}
	id := r.Task.ID
	d, _ := s.Get(id)
	before := string(d.Bytes())
	_, e = s.SetText(id, map[string]string{"Goal": "It works. Then it ships.", "Purpose": "P"}, nil)
	var te *Error
	if !errors.As(e, &te) || te.Kind != "goal-too-long" || te.Fix != "hug task set "+id+` --goal "one sentence" --details "..."` {
		t.Fatal(e)
	}
	if d, _ := s.Get(id); string(d.Bytes()) != before {
		t.Fatal("refused set wrote")
	}
	if _, e := s.SetText(id, map[string]string{"Goal": "It ships.", "Relevant details": "It works first. Then it ships."}, nil); e != nil {
		t.Fatal(e)
	}
	// A Goal already on disk, long, is read as it is and other sets work.
	long := strings.Repeat("Old goal sentence. ", 20)
	path := s.Root + "/" + d.Task.Path
	b, _ := os.ReadFile(path)
	os.WriteFile(path, []byte(strings.Replace(string(b), "## Goal\nIt ships.\n", "## Goal\n"+long+"\n", 1)), 0o644)
	s.Invalidate()
	d, e = s.Get(id)
	if e != nil || strings.TrimSpace(d.Task.Goal) != strings.TrimSpace(long) {
		t.Fatalf("%v %q", e, d.Task.Goal)
	}
	if _, e := s.SetText(id, map[string]string{"Purpose": "Why"}, nil); e != nil {
		t.Fatalf("set of another section refused over the old Goal: %v", e)
	}
}

// 0pr03h check 2: lint warns on a long Goal of an open task (advice, with
// the fix, on the Goal's line), not on a closed one, and never rewrites.
func TestLintGoalTooLong(t *testing.T) {
	body := "\nstate: todo\n---\n\n## Goal\nIt works. Then it ships.\n\n## Purpose\np\n\n## Principle\nq\n\n## Acceptance\n- [ ] a\n"
	open := Parse("q.abc123.t.md", []byte("---\nid: abc123\ntitle: T"+body))
	closed := Parse("q.def456.t.md", []byte("---\nid: def456\ntitle: T"+strings.Replace(body, "state: todo", "state: done", 1)))
	fine := Parse("q.ghi789.t.md", []byte("---\nid: ghi789\ntitle: T"+strings.Replace(body, "It works. Then it ships.", "Ship it, e.g. as v0.7.85 in main.go.", 1)))
	fs, _ := OpenFindings(Lint([]*Document{open, closed, fine}), []Task{open.Task, closed.Task, fine.Task})
	var got []Finding
	for _, f := range fs {
		if FindingCode(f.Problem) == "goal-too-long" {
			got = append(got, f)
		}
	}
	if len(got) != 1 || got[0].Path != "q.abc123.t.md" || got[0].Line != 1 || got[0].Severity() != Advice || got[0].FixLine() != `hug task set abc123 --goal "one sentence" --details "..."` {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got[0].String(), "q.abc123.t.md:1: advice [goal-too-long] Goal is 2 sentences: ") {
		t.Fatal(got[0].String())
	}
	if g := ByTask(got)[0].Fix; g != `hug task set abc123 --goal "one sentence" --details "..."` {
		t.Fatal(g)
	}
	if strings.TrimSpace(open.Task.Goal) != "It works. Then it ships." {
		t.Fatal("lint changed the Goal")
	}
}

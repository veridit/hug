package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// askedFixture is a board with four open questions and one answered:
// B (todo) asked of jhf on 2 Jan, no check; A (doing) asked of jhf on
// 3 Jan about check 1, then again on 4 Jan with no check; C asked of crab
// on 1 Jan; D's question to jhf is answered. HUG_USER is jhf.
func askedFixture(t *testing.T) (a *App, ids map[string]string) {
	t.Helper()
	a = app(t)
	a.Env["HUG_USER"] = "jhf"
	execute(t, a, "init")
	ids = map[string]string{}
	for _, n := range []string{"A", "B", "C", "D"} {
		ids[n] = execute(t, a, "new", "Task "+n, "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "first check", "--accept", "second check")["task"].(Reply)["id"].(string)
	}
	execute(t, a, "take", ids["A"], "--as", "nautilus", "--model", "m")
	at := func(day, hour int) { pinClock(t, time.Date(2026, 1, day, hour, 30, 0, 0, time.Local)) }
	at(1, 9)
	execute(t, a, "ask", ids["C"], "--to", "crab", "Is the C column wide enough?", "--as", "piglet")
	at(2, 9)
	execute(t, a, "ask", ids["B"], "--to", "jhf", "Should B ship before the release?\nSecond line stays out of the row.", "--as", "piglet")
	at(3, 10)
	execute(t, a, "ask", ids["A"], "--to", "jhf", "--check", "1", "Does check 1 hold on your machine?", "--as", "nautilus")
	at(4, 11)
	execute(t, a, "ask", ids["A"], "--to", "jhf", "And may A keep the old flag?", "--as", "nautilus")
	execute(t, a, "ask", ids["D"], "--to", "jhf", "Answered already?", "--as", "piglet")
	execute(t, a, "answer", ids["D"], "yes", "--as", "jhf")
	return a, ids
}

// fthher check 1: hug task ls --asked lists the open questions asked of
// the caller (HUG_USER), one line per question, oldest first: task id,
// check (- for none), the question's first line, who asked, since when.
func TestLsAskedListsQuestionsForCaller(t *testing.T) {
	a, ids := askedFixture(t)
	code, out := runText(a, "ls", "--asked")
	if code != 0 {
		t.Fatalf("ls --asked: exit %d\n%s", code, out)
	}
	var rows []string
	for _, l := range strings.Split(out, "\n") {
		if strings.HasPrefix(l, ids["A"]+" ") || strings.HasPrefix(l, ids["B"]+" ") || strings.HasPrefix(l, ids["C"]+" ") || strings.HasPrefix(l, ids["D"]+" ") {
			rows = append(rows, l)
		}
	}
	if len(rows) != 3 {
		t.Fatalf("want 3 question rows (B, A 1, A 2), got %d:\n%s", len(rows), out)
	}
	want := []struct{ id, check, text, by, since string }{
		{ids["B"], "-", "Should B ship before the release?", "piglet", "since 2 Jan"},
		{ids["A"], "1", "Does check 1 hold on your machine?", "nautilus", "since 3 Jan"},
		{ids["A"], "-", "And may A keep the old flag?", "nautilus", "since 4 Jan"},
	}
	for i, w := range want {
		f := strings.Fields(rows[i])
		if f[0] != w.id || f[1] != w.check || !strings.Contains(rows[i], w.text) || !strings.Contains(rows[i], "  "+w.by+"  ") || !strings.HasSuffix(rows[i], w.since) {
			t.Errorf("row %d = %q, want id %s check %s %q by %s %s", i+1, rows[i], w.id, w.check, w.text, w.by, w.since)
		}
	}
	if strings.Contains(out, "Second line stays out") || strings.Contains(out, "Answered already") || strings.Contains(out, "C column") {
		t.Fatalf("a second line, an answered question or another's question is listed:\n%s", out)
	}
	if !strings.Contains(out, "\n3 open questions for jhf, on 2 tasks\nnext: hug task answer "+ids["B"]+" 1 ") {
		t.Fatalf("no count line, or next does not answer the oldest:\n%s", out)
	}

	// --to NAME: the questions asked of NAME; --as is who the caller is.
	for _, args := range [][]string{{"ls", "--asked", "--to", "crab"}, {"ls", "--asked", "--as", "crab"}} {
		_, out = runText(a, args...)
		if !strings.Contains(out, ids["C"]+"  -  ") || !strings.Contains(out, "Is the C column wide enough?") || strings.Contains(out, "Should B ship") {
			t.Errorf("%v:\n%s", args, out)
		}
	}

	// Filters combine: --state todo keeps B's question only (A is doing).
	_, out = runText(a, "ls", "--asked", "--state", "todo")
	if !strings.Contains(out, "Should B ship") || strings.Contains(out, "Does check 1 hold") {
		t.Fatalf("ls --asked --state todo:\n%s", out)
	}

	// --json and --jsonl: the same rows as objects.
	code, out = runText(a, "ls", "--asked", "--json")
	var r struct {
		Asked []map[string]any `json:"asked"`
	}
	if e := json.Unmarshal([]byte(out), &r); e != nil || code != 0 || len(r.Asked) != 3 {
		t.Fatalf("--json: exit %d %v\n%s", code, e, out)
	}
	if q := r.Asked[1]; q["id"] != ids["A"] || q["n"] != float64(1) || q["check"] != float64(1) || q["question"] != "Does check 1 hold on your machine?" || q["by"] != "nautilus" || q["since"] != "since 3 Jan" || q["to"] != "jhf" {
		t.Fatalf("--json row 2: %v", q)
	}
	if q := r.Asked[0]; q["check"] != float64(0) || !strings.Contains(q["question"].(string), "Second line stays out") {
		t.Fatalf("--json row 1 (no check, the whole question): %v", q)
	}
	_, out = runText(a, "ls", "--asked", "--jsonl")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], `"id":"`+ids["B"]+`"`) {
		t.Fatalf("--jsonl:\n%s", out)
	}
}

// fthher: nothing asked of the caller says so and ends with the fix.
func TestLsAskedEmptySaysSo(t *testing.T) {
	a, _ := askedFixture(t)
	code, out := runText(a, "ls", "--asked", "--to", "nobody")
	if code != 0 || !strings.Contains(out, "no open questions for nobody\n") || !strings.HasSuffix(out, "fix: hug task ls --asked --to NAME\n") {
		t.Fatalf("exit %d, want 0 saying none and ending with the fix:\n%s", code, out)
	}
	code, out = runText(a, "ls", "--to", "jhf")
	if code != 2 || !strings.Contains(out, "fix: hug task ls --asked --to jhf") {
		t.Fatalf("--to without --asked: exit %d\n%s", code, out)
	}
}

// fthher: help ls, the AGENTS.md section hug setup writes and this
// repository's AGENTS.md name --asked in one phrase.
const askedPhrase = "hug task ls --asked lists the questions waiting for you."

func TestInstructionsNameLsAsked(t *testing.T) {
	t.Parallel()
	repo, err := os.ReadFile(filepath.Join("..", "..", "..", "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	flat := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, c := range []struct{ where, text string }{
		{"help ls", commandHelp("ls")},
		{"AGENTS.md section (hug setup)", agentSection},
		{"repository AGENTS.md", string(repo)},
	} {
		if !strings.Contains(flat(c.text), askedPhrase) {
			t.Errorf("%s does not say %q", c.where, askedPhrase)
		}
	}
}

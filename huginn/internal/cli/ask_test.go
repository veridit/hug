package cli

import (
	"strings"
	"testing"
)

// 41hhgn check 2: hug task states lists doing/asked with its fields; ask
// sets the stage with them; a bad --check is refused with the number of
// checks and a fix.
func TestAskSetsStageAsked(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Ask", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	_, out := runText(a, "states")
	for _, want := range []string{"asked", "asked_what", "asked_for", "asked_check", "asked_since", "asked_by", "asked_show", `hug task ask ID --to NAME [--check N] "question"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("states lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "waiting_for") || strings.Contains(out, "waiting_show") {
		t.Fatalf("states still lists waiting_for or waiting_show:\n%s", out)
	}
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	code, out := runText(a, "ask", id, "--to", "jhf", "--check", "9", "look at the rows", "--as", "nautilus")
	if code != 2 || !strings.Contains(out, "--check 9: "+id+" has 2 checks") || !strings.Contains(out, "fix: hug task ask "+id+" --to jhf --check 2 \"look at the rows\"\n") {
		t.Fatalf("--check 9 of 2: exit %d\n%s", code, out)
	}
	if code, out = runText(a, "ask", id, "look at the rows", "--as", "nautilus"); code != 2 || !strings.Contains(out, "--to NAME") {
		t.Fatalf("no --to: exit %d\n%s", code, out)
	}
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "look at the rows", "--as", "nautilus")
	file := taskFile(t, a, id)
	for _, want := range []string{`stage: "asked"`, `asked_what: "look at the rows"`, `asked_for: "jhf"`, `asked_check: "2"`, `asked_by: "nautilus"`, "asked_since: "} {
		if !strings.Contains(file, want) {
			t.Fatalf("ask did not write %q:\n%s", want, file)
		}
	}
	if _, out = runText(a, "show", id); !strings.Contains(out, "stage: asked jhf · check 2: look at the rows since ") {
		t.Fatalf("show:\n%s", out)
	}
	if code, out = runText(a, "lint", id); code != 0 && strings.Contains(out, "stage") {
		t.Fatalf("lint on an asked task: %d\n%s", code, out)
	}
}

// 41hhgn check 3: wait --on NAME refuses with the ask command; a plain wait
// still works.
func TestWaitOnRefusesWithAsk(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Wait", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	code, out := runText(a, "wait", id, "--on", "jhf", "approve it", "--as", "nautilus")
	if code != 2 || !strings.Contains(out, "a question to a person is hug task ask") || !strings.Contains(out, "fix: hug task ask "+id+" --to jhf \"approve it\"\n") {
		t.Fatalf("wait --on: exit %d\n%s", code, out)
	}
	if file := taskFile(t, a, id); strings.Contains(file, "waiting_") || strings.Contains(file, "asked_") {
		t.Fatalf("a refused wait wrote:\n%s", file)
	}
	execute(t, a, "wait", id, "CI run 7", "--as", "nautilus")
	if file := taskFile(t, a, id); !strings.Contains(file, `stage: "waiting"`) || !strings.Contains(file, `waiting_on: "CI run 7"`) {
		t.Fatalf("plain wait:\n%s", file)
	}
}

// 41hhgn check 8 (storage, CLI and the asker's view): a 3-line answer with
// a quote and a colon, read from stdin, is kept line for line in History,
// and show prints every line.
func TestMultiLineAnswerKeepsItsLines(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Lines", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "1", "which rows?\nand in what order?", "--as", "nautilus")
	answer := "the open ones: \"doing\" first\nthen todo\n  indented: kept"
	a.In = strings.NewReader(answer + "\n")
	execute(t, a, "answer", id, "-", "--as", "jhf")
	file := taskFile(t, a, id)
	want := "answered asked (check 1), by jhf, for nautilus (doing/asked -> doing/active):\n  > which rows?\n  > and in what order?\n  the open ones: \"doing\" first\n  then todo\n    indented: kept\n"
	if !strings.Contains(file, want) {
		t.Fatalf("History lacks the block:\n%s\nfile:\n%s", want, file)
	}
	if strings.Contains(file, "\nasked_") || !strings.Contains(file, `stage: "active"`) {
		t.Fatalf("the asked fields stayed:\n%s", file)
	}
	r := execute(t, a, "show", id)
	ans, _ := r["task"].(Reply)["answer"].(map[string]any)
	if ans["text"] != answer || ans["question"] != "which rows?\nand in what order?" || ans["check"] != float64(1) && ans["check"] != 1 {
		t.Fatalf("answer read back: %#v", ans)
	}
	_, out := runText(a, "show", id)
	for _, l := range strings.Split(answer, "\n") {
		if !strings.Contains(out, l+"\n") {
			t.Fatalf("show lacks the line %q:\n%s", l, out)
		}
	}
	// "-" with nothing on stdin is an empty answer, refused with a fix.
	execute(t, a, "ask", id, "--to", "jhf", "again?", "--as", "nautilus")
	a.In = strings.NewReader("")
	if code, out := runText(a, "answer", id, "-", "--as", "jhf"); code != 2 || !strings.Contains(out, "answer needs the answer") {
		t.Fatalf("empty stdin: %d\n%s", code, out)
	}
}

// 41hhgn (crab 00:44, Statbus's agent: "show --json doesn't return
// waiting_for"): show --json puts the stage's own fields in an object
// named after the stage, beside "stage", and the flat fields map no longer
// repeats them.
func TestShowJSONStageObject(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Ask", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "--check", "2", "which rows?\nin what order?", "--as", "nautilus")
	tk := execute(t, a, "show", id)["task"].(Reply)
	asked, _ := tk["asked"].(Reply)
	if tk["stage"] != "asked" || asked["what"] != "which rows?\nin what order?" || asked["for"] != "jhf" || asked["check"] != 2 || asked["by"] != "nautilus" || asked["since"] == nil {
		t.Fatalf("show --json asked: stage %v, asked %#v", tk["stage"], tk["asked"])
	}
	if f, _ := tk["fields"].(map[string]any); f["asked_what"] != nil || f["asked_for"] != nil || f["doing_since"] == nil {
		t.Fatalf("flat fields repeat the stage's or lost the rest: %#v", tk["fields"])
	}
	execute(t, a, "resume", id)
	execute(t, a, "wait", id, "CI run 9", "--as", "nautilus")
	tk = execute(t, a, "show", id)["task"].(Reply)
	if w, _ := tk["waiting"].(Reply); tk["stage"] != "waiting" || w["on"] != "CI run 9" || w["asked_by"] != "nautilus" || tk["asked"] != nil {
		t.Fatalf("show --json waiting: %#v / asked %#v", tk["waiting"], tk["asked"])
	}
	execute(t, a, "resume", id)
	execute(t, a, "review", id, "--by", "jhf")
	if r, _ := execute(t, a, "show", id)["task"].(Reply)["review"].(Reply); r["by"] != "jhf" {
		t.Fatalf("show --json review: %#v", r)
	}
	todo := execute(t, a, "new", "Design", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "design", todo, "this or that?", "--for", "jhf")
	if d, _ := execute(t, a, "show", todo)["task"].(Reply)["design"].(Reply); d["question"] != "this or that?" || d["for"] != "jhf" {
		t.Fatalf("show --json design: %#v", d)
	}
	// The text show still prints the stage line from the object.
	if _, out := runText(a, "show", todo); !strings.Contains(out, "this or that?") {
		t.Fatalf("text show lost the design line:\n%s", out)
	}
}

// qsqhpe check 4: a ticked check cannot be asked about (a tick is the
// answer): hug task ask --check N on a ticked check, now or --after, is
// refused with exit 2 and a fix: line, and writes nothing.
func TestAskOnTickedCheckRefused(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Ask", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "check", id, "1")
	before := taskFile(t, a, id)
	for _, args := range [][]string{
		{"ask", id, "--to", "jhf", "--check", "1", "does it hold?", "--as", "nautilus"},
		{"ask", id, "--to", "jhf", "--check", "1", "--after", "2099-01-01", "does it hold?", "--as", "nautilus"},
	} {
		code, out := runText(a, args...)
		if code != 2 || !strings.Contains(out, "check 1 is already ticked: ask about an open check, or add one with hug task set "+id+" --accept ...") ||
			!strings.Contains(out, "fix: hug task ask "+id+" --to jhf --check 2 \"does it hold?\"\n") {
			t.Fatalf("%v: exit %d\n%s", args, code, out)
		}
	}
	if after := taskFile(t, a, id); after != before {
		t.Fatalf("a refused ask wrote:\n%s", after)
	}
	// Every check ticked: the fix adds one.
	execute(t, a, "check", id, "2")
	code, out := runText(a, "ask", id, "--to", "jhf", "--check", "2", "does it hold?", "--as", "nautilus")
	if code != 2 || !strings.Contains(out, "fix: hug task set "+id+" --accept \"does it hold?\"\n") {
		t.Fatalf("all ticked: exit %d\n%s", code, out)
	}
}

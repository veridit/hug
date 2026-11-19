package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/task"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	b, e := c.CombinedOutput()
	if e != nil {
		t.Fatalf("git %v: %s %v", args, b, e)
	}
	return strings.TrimSpace(string(b))
}
func app(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	git(t, dir, "config", "user.name", "Test")
	git(t, dir, "config", "user.email", "test@example.invalid")
	git(t, dir, "commit", "--allow-empty", "-m", "init")
	// The App's own environment, never the process's: tests that use app run
	// in parallel (0sbvnq), which a process-wide setenv rules out.
	return &App{Cwd: dir, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{},
		// No machine's real session list in tests (v3k6cp stale-owner), nor
		// the jcode instances its ~/.zshrc names (9ahvgt).
		Env: map[string]string{"HUG_USER": "tester", "HUG_MODEL": "test-model", "JCODE_SESSION_ID": "", "JCODE_HOME": filepath.Join(dir, ".no-jcode"), "HOME": filepath.Join(dir, ".no-home")}}
}
func execute(t *testing.T, a *App, args ...string) Reply {
	t.Helper()
	r, e := a.Execute(context.Background(), args)
	if e != nil {
		t.Fatalf("%v: %v", args, e)
	}
	if r["next"] == "" {
		t.Fatalf("missing next: %v", r)
	}
	return r
}
func expectCode(t *testing.T, a *App, want int, args ...string) {
	t.Helper()
	_, e := a.Execute(context.Background(), args)
	if e == nil {
		t.Fatalf("%v succeeded", args)
	}
	code, p, fix := ErrorDetails(e)
	if code != want || p == "" || fix == "" || strings.Contains(p, "\n") {
		t.Fatalf("%v: got %d %q %q, want %d", args, code, p, fix, want)
	}
}
func TestLifecycleJSONContract(t *testing.T) {
	today := pinClock(t, lastTick) // the done's note is dated by the board clock (jsyksd)
	a := app(t)
	execute(t, a, "init")
	expectCode(t, a, 4, "next")
	execute(t, a, "rebalance")
	r := execute(t, a, "new", "API", "--goal", "API works", "--purpose", "P", "--principle", "Q", "--accept", "First", "--accept", "Second", "--type", "bug")
	id := r["task"].(Reply)["id"].(string)
	projection := r["task"].(Reply)
	if projection["goal"] != "API works" {
		t.Fatalf("goal contract: %q", projection["goal"])
	}
	acceptanceJSON, _ := json.Marshal(projection["acceptance"])
	if string(acceptanceJSON) != `[{"checked":false,"state":"open","text":"First"},{"checked":false,"state":"open","text":"Second"}]` {
		t.Fatalf("acceptance contract: %s", acceptanceJSON)
	}
	dep := execute(t, a, "new", "Form", "--goal", "Form works", "--purpose", "P", "--principle", "Q", "--accept", "UI", "--dep", id)
	form := dep["task"].(Reply)["id"].(string)
	expectCode(t, a, 3, "take", form)
	// done only from doing: an untaken task's fix is the take (6ephqk).
	if code, out := runText(a, "done", id); code != 2 || !strings.Contains(out, "fix: hug task take "+id+" --model MODEL") {
		t.Fatalf("done on todo: %d %s", code, out)
	}
	execute(t, a, "take", id)
	execute(t, a, "take", id)
	expectCode(t, a, 3, "done", id)
	execute(t, a, "next", "--mine")
	execute(t, a, "check", id, "1")
	execute(t, a, "check", id, "1")
	execute(t, a, "note", id, "Evidence here")
	execute(t, a, "check", id, "2")
	execute(t, a, "done", id)
	execute(t, a, "done", id)
	r = execute(t, a, "show", id)
	task := r["task"].(Reply)
	// The plain done left one unanswered line (2k68eg); the repeat no second.
	if task["notes"] != "Evidence here\nReflection (unanswered, "+today+", tester)" {
		t.Fatal(task)
	}
	r = execute(t, a, "show", form)
	deps := r["task"].(Reply)["dependencies"].([]string)
	if len(deps) != 1 || deps[0] != id {
		t.Fatal(r)
	}
	execute(t, a, "set", form, "--title", "New Form", "--priority", "high", "--label", "ui")
	execute(t, a, "mv", form, "auth")
	execute(t, a, "move", form, "--top")
	execute(t, a, "rebalance")
	expectCode(t, a, 2, "drop", form) // a drop needs its reason (6ephqk)
	execute(t, a, "drop", form, "superseded by the new form")
	execute(t, a, "drop", form, "superseded by the new form")
	r = execute(t, a, "ls")
	if len(r["tasks"].([]Reply)) != 2 {
		t.Fatal(r)
	}
	r = execute(t, a, "lint")
	b, _ := json.Marshal(r["findings"])
	if string(b) != "[]" {
		t.Fatal(string(b))
	}
	git(t, filepath.Join(a.Cwd, ".hug", "tasks"), "diff", "--exit-code")
	if got := git(t, filepath.Join(a.Cwd, ".hug", "tasks"), "status", "--porcelain"); got != "" {
		t.Fatal(got)
	}
	if e := os.Remove(cache.File(filepath.Join(a.Cwd, ".git"))); e != nil {
		t.Fatal(e) // the cache must exist here, or this checks nothing
	}
	r = execute(t, a, "ls")
	if len(r["tasks"].([]Reply)) != 2 {
		t.Fatal(r)
	}
}
func TestUsageAndText(t *testing.T) {
	t.Parallel()
	a := app(t)
	for _, args := range [][]string{{"nex"}, {"next", "--typo", "bug"}, {"ls", "--state", "don"}, {"next", "--type", "bugs"}, {"new", "x"}, {"check", "abc", "zero"}} {
		expectCode(t, a, 2, args...)
	}
	if len(strings.Split(strings.TrimSpace(Help), "\n")) >= 40 {
		t.Fatal("help too long")
	}
	for _, args := range [][]string{nil, {"help"}, {"types"}, {"init"}, {"new", "A", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C"}, {"ls"}, {"show", "a"}} {
		a.Out = &bytes.Buffer{}
		a.Err = &bytes.Buffer{}
		if code := a.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v code %d: %s", args, code, a.Err)
		}
		lines := strings.Split(strings.TrimSpace(a.Out.(*bytes.Buffer).String()), "\n")
		if !strings.HasPrefix(lines[len(lines)-1], "next: hug task ") {
			t.Fatal(lines)
		}
	}
}
func TestIdentityAndCommandHelp(t *testing.T) {
	t.Parallel()
	a := app(t)
	// No name and no model: --mine has nobody to mean. git user.name
	// ("Test", set by app) is never used (et85h1).
	a.Env["HUG_USER"], a.Env["HUG_MODEL"] = "", ""
	expectCode(t, a, 2, "next", "--mine")
	r := execute(t, a, "help", "new")
	if !strings.Contains(r["help"].(string), "--dep") || !strings.Contains(r["help"].(string), "--group") {
		t.Fatal(r)
	}
	execute(t, a, "next", "--help")
	expectCode(t, a, 2, "help", "nex")
}

func TestRejectedSetIsUnchanged(t *testing.T) {
	t.Parallel()
	a := app(t)
	r := execute(t, a, "new", "Keep title", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")
	id := r["task"].(Reply)["id"].(string)
	expectCode(t, a, 2, "set", id, "--state", "doing", "--title", "")
	expectCode(t, a, 2, "set", id, "--type", "bugs")
	_, groupError := a.Execute(context.Background(), []string{"mv", "keep-title", "../bad"})
	_, _, groupFix := ErrorDetails(groupError)
	if groupFix != "hug task mv "+id+" auth" {
		t.Fatal(groupFix)
	}
	r = execute(t, a, "show", id)
	if r["task"].(Reply)["title"] != "Keep title" || r["task"].(Reply)["state"] != "todo" {
		t.Fatal(r)
	}
	a.Out = &bytes.Buffer{}
	a.Err = &bytes.Buffer{}
	if a.Run(context.Background(), []string{"unknown\ncommand"}) != 2 {
		t.Fatal("usage code")
	}
	if len(strings.Split(strings.TrimSpace(a.Err.(*bytes.Buffer).String()), "\n")) != 2 {
		t.Fatal(a.Err)
	}
}

func TestDependencyRecoveryAndCycle(t *testing.T) {
	t.Parallel()
	a := app(t)
	first := execute(t, a, "new", "API", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	second := execute(t, a, "new", "Form", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	execute(t, a, "dep", second, "--add", "api")
	repeated := execute(t, a, "dep", second, "--add", first)
	if repeated["changed"] != false {
		t.Fatal(repeated)
	}
	expectCode(t, a, 3, "dep", first, "--add", second)
	_, cycleError := a.Execute(context.Background(), []string{"dep", first, "--add", second})
	_, problem, fix := ErrorDetails(cycleError)
	if !strings.Contains(problem, "dependency cycle:") || !strings.Contains(fix, " --remove ") {
		t.Fatal(problem, fix)
	}
	expectCode(t, a, 2, "dep", second, "--remove", first, "--add", "missing-task")
	r := execute(t, a, "show", second)
	deps := r["task"].(Reply)["dependencies"].([]string)
	if len(deps) != 1 || deps[0] != first {
		t.Fatal(r)
	}
	if dirty := git(t, filepath.Join(a.Cwd, ".hug", "tasks"), "status", "--porcelain"); dirty != "" {
		t.Fatal(dirty)
	}
	execute(t, a, "dep", second, "--remove", first)
	execute(t, a, "dep", second, "--remove", first)
	expectCode(t, a, 2, "move", second)
}

func TestJSONErrors(t *testing.T) {
	t.Parallel()
	a := app(t)
	if code := a.Run(context.Background(), []string{"ls", "--state", "don", "--json"}); code != 2 {
		t.Fatal(code)
	}
	var r Reply
	if e := json.Unmarshal(a.Out.(*bytes.Buffer).Bytes(), &r); e != nil {
		t.Fatal(e)
	}
	if r["next"] == nil || r["fix"] == nil {
		t.Fatal(r)
	}
}

// board prints todo/doing/done side by side, marks blocked tasks and keeps
// done short unless --all; the JSON carries the same columns.
func TestBoardColumns(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	first := execute(t, a, "new", "First quartz", "--goal", "g", "--purpose", "P", "--principle", "Q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "new", "Second quartz", "--goal", "g", "--purpose", "P", "--principle", "Q", "--accept", "a", "--dep", first)
	for i := 0; i < 7; i++ {
		id := execute(t, a, "new", "Old quartz", "--goal", "g", "--purpose", "P", "--principle", "Q", "--accept", "a")["task"].(Reply)["id"].(string)
		execute(t, a, "check", id, "1")
		execute(t, a, "take", id)
		execute(t, a, "done", id)
	}
	execute(t, a, "take", first)
	r := execute(t, a, "board")
	cols := r["board"].([]BoardColumn)
	if len(cols) != 3 || cols[0].Total != 1 || cols[1].Total != 1 || cols[2].Total != 7 || len(cols[2].Cards) != 5 || cols[2].Hidden != 2 {
		t.Fatalf("columns: %+v", cols)
	}
	if !cols[0].Cards[0].Blocked {
		t.Fatal("dependent todo not marked blocked")
	}
	if all := execute(t, a, "board", "--all")["board"].([]BoardColumn); len(all[2].Cards) != 7 || all[2].Hidden != 0 {
		t.Fatalf("--all: %+v", all[2])
	}
	var out bytes.Buffer
	a.Out = &out
	a.Env["COLUMNS"] = "90"
	if code := a.Run(context.Background(), []string{"board"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	text := out.String()
	for _, want := range []string{"TODO (1)", "DOING (1)", "DONE (7)", "! Second quartz", "2 more (--all)"} {
		if !strings.Contains(text, want) {
			t.Fatalf("board text missing %q:\n%s", want, text)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if len([]rune(line)) > 90 {
			t.Fatalf("line wider than COLUMNS: %q", line)
		}
	}
}

// rx74bp: Jørgen's edit, the exact text, typed into Depends on through
// hug task edit. lint names the line, says it is not a dependency, prints
// a fix per finding, and next: repairs the first finding (text and JSON);
// the task is not blocked.
func TestLintNextFixesFirstFinding(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	r := execute(t, a, "new", "Recall nudges", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")
	id := r["task"].(Reply)["id"].(string)
	other := execute(t, a, "new", "Other", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	clean := execute(t, a, "lint")
	if clean["next"] != "hug task take "+id {
		t.Fatalf("clean lint keeps the usual next: %v", clean)
	}
	useEditor(t, a, "## Depends on\n", "## Depends on\nIs there a task:asdfasdf or ticket:asdlfkjasldkfj?\n")
	edited := execute(t, a, "edit", id)
	if edited["next"] != "hug task edit "+id {
		t.Fatalf("edit next: %v", edited)
	}
	for _, args := range [][]string{{"lint"}, {"lint", "--json"}, {"lint", id}} {
		out := &bytes.Buffer{}
		a.Out, a.Err = out, &bytes.Buffer{}
		if code := a.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v: %d %s", args, code, a.Err)
		}
		text := out.String()
		t.Logf("%v:\n%s", args, text)
		if args[len(args)-1] == "--json" {
			var m map[string]any
			if e := json.Unmarshal(out.Bytes(), &m); e != nil {
				t.Fatal(e)
			}
			f := m["findings"].([]any)[0].(map[string]any)
			if m["next"] != "hug task edit "+id || f["code"] != "not-a-dependency" || f["fix"] != "hug task edit "+id {
				t.Fatalf("json: %s", text)
			}
			continue
		}
		want := `[not-a-dependency] not a dependency: "Is there a task:asdfasdf or ticket:asdlfkjasldkfj?" (for no dependencies, leave the section empty or remove it; a dependency line is "- <task id>", or run hug task dep ` + id + ` --add REF; prose belongs in Notes)` + "\nnext: hug task edit " + id + "\n"
		if len(args) == 1 {
			// The overview (fkpyg3): one line per task, next is its fix.
			want = "  " + id + "  not-a-dependency  -> hug task lint " + id + "\nnext: hug task edit " + id + "\n"
		}
		if !strings.HasSuffix(text, want) || strings.Contains(text, other) || strings.Contains(text, "dangling") {
			t.Fatalf("got:\n%s\nwant suffix:\n%s", text, want)
		}
	}
	a.Out = &bytes.Buffer{}
	shown := execute(t, a, "show", id)["task"].(Reply)
	if deps := shown["dependencies"].([]string); len(deps) != 0 {
		t.Fatalf("prose became a dependency: %v", deps)
	}
	if next := execute(t, a, "next")["task"].(Reply)["id"]; next != id {
		t.Fatalf("prose blocked the task: next is %v", next)
	}
}

// 6neax1: lint is never silent, has a summary by code, lists the codes for
// --code alone, rejects an unknown code with the list, and reads "help" as
// asking for help, before or after flags. Errors end in a runnable fix.
func TestLintNeverSilentSummaryCodesHelp(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	run := func(args ...string) (int, string, string) {
		out, errb := &bytes.Buffer{}, &bytes.Buffer{}
		a.Out, a.Err = out, errb
		code := a.Run(context.Background(), args)
		return code, out.String(), errb.String()
	}
	// An empty board, then a clean one: "no findings" and a next: line.
	if code, out, _ := run("lint"); code != 0 || !strings.HasPrefix(out, "no findings\nnext: hug task ") {
		t.Fatalf("empty board: %d %q", code, out)
	}
	id := execute(t, a, "new", "Clean", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	if code, out, _ := run("lint"); code != 0 || out != "no findings\nnext: hug task take "+id+"\n" {
		t.Fatalf("clean board: %d %q", code, out)
	}
	// Two findings of two codes: a long title (set accepts it; lint and
	// the set reply name it) and a Principle removed by hand.
	long := execute(t, a, "new", "Long", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	set := execute(t, a, "set", long, "--title", strings.Repeat("Long title ", 9))
	if l, ok := set["lint"].(*TaskLint); !ok || l.Fix != "hug task set "+long+` --title "..."` {
		t.Fatalf("set reply lint: %+v", set["lint"])
	}
	useEditor(t, a, "## Principle\nQ\n", "")
	execute(t, a, "edit", long)
	code, out, _ := run("lint", "--summary")
	t.Log("--summary:\n" + out)
	if code != 0 || !strings.HasPrefix(out, "2 findings (0 broken) in 2 codes:\n") || strings.Contains(out, "\t") ||
		!strings.Contains(out, "  1  advice  long-title       the title is longer than 80 characters   hug task lint --code long-title\n") ||
		!strings.Contains(out, "  1  advice  missing-section  an open task lacks Purpose or Principle  hug task lint --code missing-section\n") ||
		!strings.HasSuffix(out, "next: hug task lint --code long-title\n") {
		t.Fatalf("summary: %q", out)
	}
	r := execute(t, a, "lint", "--summary")
	if s := r["summary"].(LintSummary); s.Total != 2 || len(s.Codes) != 2 || len(r["findings"].([]task.Finding)) != 2 {
		t.Fatalf("json summary: %+v", r)
	}
	b, _ := json.Marshal(r)
	if !strings.Contains(string(b), `"summary":{"total":2,"broken":0,"codes":[{"code":"long-title","count":1,"meaning":"the title is longer than 80 characters","severity":"advice","list":"hug task lint --code long-title"}`) {
		t.Fatalf("json: %s", b)
	}
	// --code alone and --code help list every code with its count.
	for _, args := range [][]string{{"lint", "--code"}, {"lint", "--code", "help"}, {"lint", "--code", "--json"}} {
		code, out, errs := run(args...)
		if code != 0 || errs != "" {
			t.Fatalf("%v: %d %s", args, code, errs)
		}
		if args[len(args)-1] == "--json" {
			if !strings.Contains(out, `"code":"not-a-dependency","count":0`) {
				t.Fatalf("%v: %s", args, out)
			}
			continue
		}
		if !strings.HasPrefix(out, "finding codes (2 findings on this board):\n  1  advice  long-title ") || !strings.Contains(out, "  0  broken  not-a-dependency ") || strings.Count(out, "\n") != len(task.Codes)+2 {
			t.Fatalf("%v: %q", args, out)
		}
	}
	// An unknown code says so, lists the valid ones, and its fix runs.
	code, _, errs := run("lint", "--code", "nope")
	if code != 2 || !strings.HasPrefix(errs, "unknown code nope; closest: ") || !strings.Contains(errs, "valid: frontmatter, ") || !strings.HasSuffix(errs, "\nfix: hug task lint --code\n") {
		t.Fatalf("unknown code: %d %q", code, errs)
	}
	// help as a positional, before or after flags, is lint's help.
	for _, args := range [][]string{{"lint", "help"}, {"lint", "help", "--code"}, {"lint", "--code", "x", "help"}, {"lint", "--summary", "help"}, {"lint", "--help"}} {
		code, out, errs := run(args...)
		if code != 0 || !strings.HasPrefix(out, "hug task lint [ID] [--summary] [--code [CODE]]\n") || errs != "" {
			t.Fatalf("%v: %d %q %q", args, code, out, errs)
		}
	}
	// "new help" is still a title, not a request for help.
	if r := execute(t, a, "new", "help", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C"); r["task"].(Reply)["title"] != "help" {
		t.Fatalf("new help: %v", r)
	}
	// Flag mistakes end in a fix that runs: the command's help.
	code, _, errs = run("lint", "--sumary")
	if code != 2 || !strings.HasSuffix(errs, "\nfix: hug task lint --help\n") || !strings.Contains(errs, "closest: --summary") {
		t.Fatalf("typo: %d %q", code, errs)
	}
}

// More than lintBound findings: summary, the first lintShown, how to see all
// (--json; --all went with t01kqm).
func TestLintBoundedOutput(t *testing.T) {
	t.Parallel()
	var fs []task.Finding
	for i := 0; i < 35; i++ {
		fs = append(fs, task.Finding{Path: fmt.Sprintf("%03d.k7m2q%c.t.md", i, "0123456789abcdefghjkmnpqrstvwxyz"[i%32]), Line: 1, Problem: "Acceptance requires at least one item"})
	}
	fs = append(fs, task.Finding{Path: "x.zzzzzz.t.md", Line: 1, Problem: "missing Goal"})
	write := func(mode, code string) string {
		var b bytes.Buffer
		r := lintReply(Reply{}, fs, mode, code)
		writeFindings(&b, r["findings"].([]task.Finding), r["summary"].(LintSummary))
		return b.String()
	}
	out := write("", "")
	t.Log("\n" + out)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	// The overview (fkpyg3): totals per code, broken codes first however
	// few (pbh9xq), then one line per task, the first lintShown tasks when
	// there are more than lintBound.
	if lines[0] != "36 findings (1 broken) on 36 tasks, in 2 codes:" || lines[1] != "   1  broken  missing-goal   no Goal section, or it is empty  hug task lint --code missing-goal" ||
		lines[2] != "  35  advice  no-acceptance  no acceptance item (- [ ] ...)   hug task lint --code no-acceptance" || lines[3] != "first 10 tasks:" ||
		lines[4] != "  k7m2q0  no-acceptance  -> hug task lint k7m2q0" ||
		len(lines) != 4+lintShown+1 || lines[len(lines)-1] != "10 of 36 tasks shown; one kind: hug task lint --code CODE; every one: hug task lint --json" {
		t.Fatalf("bounded:\n%s", out)
	}
	if one := write("", "no-acceptance"); !strings.HasSuffix(one, "10 of 36 shown; every one: hug task lint --code no-acceptance --json\n") || strings.Contains(one, " in 2 codes") {
		t.Fatalf("--code bounded:\n%s", one)
	}
}

// t01kqm: lint judges open tasks only. A done or dropped task gets no
// finding, broken or advice, and nothing counts it ("N on closed tasks
// hidden" is gone, so is --all); hug task lint ID on one says it is closed.
// A file hug cannot read as a task still shows: nothing says it is closed.
func TestLintJudgesOpenTasksOnly(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	run := func(args ...string) string {
		out := &bytes.Buffer{}
		a.Out, a.Err = out, &bytes.Buffer{}
		if code := a.Run(context.Background(), args); code != 0 {
			t.Fatalf("%v: %d\n%s", args, code, out)
		}
		return out.String()
	}
	// Broken (not-a-dependency) on a task, then the task is done.
	closedBroken := execute(t, a, "new", "Closed one", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	useEditor(t, a, "## Depends on\n", "## Depends on\nNothing\n")
	execute(t, a, "edit", closedBroken)
	execute(t, a, "check", closedBroken, "1")
	execute(t, a, "take", closedBroken)
	execute(t, a, "done", closedBroken)
	// Advice (section-order) on a task that is then dropped.
	closedAdvice := execute(t, a, "new", "Closed advice", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	m, _ := globID(filepath.Join(a.Cwd, TaskDir), closedAdvice)
	b, _ := os.ReadFile(m[0])
	text := strings.Replace(string(b), "\n## Depends on\n", "\n", 1)
	if err := os.WriteFile(m[0], []byte(strings.Replace(text, "\n## Purpose\n", "\n## Depends on\n\n## Purpose\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(a.Cwd, TaskDir), "commit", "-qam", "out of order")
	execute(t, a, "drop", closedAdvice, "not needed")
	// A done task with an open check, as an import or a hand edit leaves it.
	doneOpen := execute(t, a, "new", "Done by hand", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	m, _ = globID(filepath.Join(a.Cwd, TaskDir), doneOpen)
	b, _ = os.ReadFile(m[0])
	if err := os.WriteFile(m[0], []byte(strings.Replace(string(b), "state: todo", "state: done", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(a.Cwd, TaskDir), "commit", "-qam", "done by hand")

	if out := run("lint"); !strings.HasPrefix(out, "no findings\nnext: ") {
		t.Fatalf("closed tasks were judged:\n%s", out)
	}
	if out := run("lint", "--summary"); strings.Contains(out, "hidden") || strings.Contains(out, "done-unchecked") {
		t.Fatalf("summary:\n%s", out)
	}
	for _, id := range []string{closedBroken, closedAdvice, doneOpen} {
		state := "done"
		if id == closedAdvice {
			state = "dropped"
		}
		if out := run("lint", id); out != id+" is "+state+": lint judges open tasks only\nnext: hug task show "+id+"\n" {
			t.Fatalf("lint %s:\n%s", id, out)
		}
	}
	r := execute(t, a, "lint")
	jb, _ := json.Marshal(r)
	if strings.Contains(string(jb), "hidden") || len(r["findings"].([]task.Finding)) != 0 {
		t.Fatalf("json: %s", jb)
	}
	// --all is gone.
	if code, out := runText(a, "lint", "--all"); code != 2 || !strings.Contains(out, "--all") {
		t.Fatalf("lint --all: exit %d\n%s", code, out)
	}
	// Open tasks: advice and broken both shown, broken first.
	openLong := execute(t, a, "new", "Open advice", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	execute(t, a, "set", openLong, "--title", strings.Repeat("Long title ", 9)) // new refuses it (fkpyg3); set keeps it as advice
	openBroken := execute(t, a, "new", "Open broken", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	execute(t, a, "edit", openBroken)
	out := run("lint")
	iBroken, iAdvice := strings.Index(out, "  "+openBroken+"  not-a-dependency  -> "), strings.Index(out, "  "+openLong+"  long-title  -> ")
	if iBroken < 0 || iAdvice < 0 || iAdvice < iBroken || strings.Contains(out, closedAdvice) || strings.Contains(out, closedBroken) || strings.Contains(out, doneOpen) {
		t.Fatalf("open tasks:\n%s", out)
	}
	if !strings.HasPrefix(out, "2 findings (1 broken) on 2 tasks, in 2 codes:\n") {
		t.Fatalf("counts:\n%s", out)
	}
	if d := run("lint", openLong); !strings.Contains(d, ": advice [long-title] ") {
		t.Fatalf("open advice, detail:\n%s", d)
	}
	// A file hug cannot read as a task shows, closed or not unknown.
	if err := os.WriteFile(filepath.Join(a.Cwd, TaskDir, "zz.qqqqqq.garbage.md"), []byte("no header at all\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, filepath.Join(a.Cwd, TaskDir), "add", "-A")
	git(t, filepath.Join(a.Cwd, TaskDir), "commit", "-qm", "garbage")
	if out := run("lint"); !strings.Contains(out, "zz.qqqqqq.garbage.md") && !strings.Contains(out, "qqqqqq") {
		t.Fatalf("an unreadable file is not shown:\n%s", out)
	}
}

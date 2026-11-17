package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// 89y0b7 check 2 (41hhgn: ask --to writes asked_for): a person lint can check; lint
// flags a name that is no listed person and no agent on the board, with a
// fix that adds them; hug task people add writes hug.toml [people] and
// lint is quiet after it.
func TestPeopleAndWaitingFor(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "ask", id, "--to", "jhf", "look at the colours", "--as", "nautilus")
	file := taskFile(t, a, id)
	if !strings.Contains(file, `asked_for: "jhf"`) || !strings.Contains(file, `asked_what: "look at the colours"`) {
		t.Fatalf("ask --to did not write asked_for:\n%s", file)
	}
	code, out := runText(a, "lint")
	want := "hug task people add jhf EMAIL"
	if !strings.Contains(out, "unknown-person") {
		t.Fatalf("lint did not flag an unknown person (exit %d):\n%s", code, out)
	}
	if code, out = runText(a, "lint", id); !strings.Contains(out, want) || !strings.Contains(out, "asked_for jhf is not in hug.toml [people]") {
		t.Fatalf("lint %s lacks %q:\n%s", id, want, out)
	}
	// An agent on the board is known: asking nautilus is fine.
	execute(t, a, "ask", id, "--to", "nautilus", "review my diff", "--as", "nautilus")
	if _, out = runText(a, "lint"); !strings.HasPrefix(out, "no findings") {
		t.Fatalf("waiting on an agent that works here was flagged:\n%s", out)
	}
	// Add the person; lint is quiet.
	execute(t, a, "ask", id, "--to", "jhf", "look at the colours", "--as", "nautilus")
	if code, out = runText(a, "people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld"); code != 0 {
		t.Fatalf("people add: %d\n%s", code, out)
	}
	if code, out = runText(a, "people", "add", "jhf", "12345+jhf@users.noreply.github.com"); code != 0 {
		t.Fatalf("people add a second email: %d\n%s", code, out)
	}
	toml, err := os.ReadFile(filepath.Join(a.Cwd, TaskDir, task.ConfigName))
	if err != nil {
		t.Fatal(err)
	}
	if string(toml) != "layout = 2\n\n[people.jhf]\nname = \"Jørgen H. Fjeld\"\nemail = [\"12345+jhf@users.noreply.github.com\", \"jorgen@veridit.no\"]\n" {
		t.Fatalf("hug.toml:\n%s", toml)
	}
	if _, out = runText(a, "lint"); !strings.HasPrefix(out, "no findings") {
		t.Fatalf("lint after people add:\n%s", out)
	}
	// The listing names them; one email belongs to one person.
	if code, out = runText(a, "people"); code != 0 || !strings.Contains(out, "jhf  Jørgen H. Fjeld  12345+jhf@users.noreply.github.com, jorgen@veridit.no") {
		t.Fatalf("people: %d\n%s", code, out)
	}
	if code, out = runText(a, "people", "add", "jorgen", "jorgen@veridit.no"); code != 2 || !strings.Contains(out, "already jhf's") {
		t.Fatalf("an email given to a second person: %d\n%s", code, out)
	}
	// Withdrawn and waiting on CI instead: nobody is asked.
	execute(t, a, "resume", id)
	execute(t, a, "wait", id, "CI run 9", "--as", "nautilus")
	if file = taskFile(t, a, id); strings.Contains(file, "\nasked_for:") || strings.Contains(file, "\nwaiting_for:") {
		t.Fatalf("a wait on CI kept a person:\n%s", file)
	}
}

// 89y0b7: hug task people offers an entry per commit author no entry
// covers, emails grouped by author name, as a command; setup has the same
// step.
func TestPeopleOffersFromCommitLog(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	for _, c := range [][2]string{{"Jørgen H. Fjeld", "jorgen@veridit.no"}, {"Jørgen H. Fjeld", "12345+jhf@users.noreply.github.com"}, {"Ada", "ada@example.org"}} {
		cmd := exec.Command("git", "-C", a.Cwd, "commit", "-q", "--allow-empty", "-m", "c")
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME="+c[0], "GIT_AUTHOR_EMAIL="+c[1], "GIT_COMMITTER_NAME="+c[0], "GIT_COMMITTER_EMAIL="+c[1])
		if b, e := cmd.CombinedOutput(); e != nil {
			t.Fatalf("commit: %v %s", e, b)
		}
	}
	code, out := runText(a, "people")
	for _, want := range []string{
		`hug task people add ada ada@example.org --name "Ada"`,
		`hug task people add jorgen 12345+jhf@users.noreply.github.com --name "Jørgen H. Fjeld" && hug task people add jorgen jorgen@veridit.no`,
	} {
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("people lacks %q (exit %d):\n%s", want, code, out)
		}
	}
	r := execute(t, a, "setup")
	found := false
	for _, s := range r["setup"].([]SetupStep) {
		if s.Step == "people" {
			found = true
			if s.Done || !strings.HasPrefix(s.Do, "hug task people add ") {
				t.Fatalf("setup people step: %+v", s)
			}
		}
	}
	if !found {
		t.Fatal("setup has no people step")
	}
}

// 89y0b7 with asr4t2 and 41hhgn: hug upgrade moves an old waiting_on
// "NAME: what" with NAME a listed person, and a wait with waiting_for, to
// stage asked, and says so; "CI run: 9" stays a wait.
func TestUpgradeMovesWaitingFor(t *testing.T) {
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	ids := map[string]string{}
	// One doing task per agent (n2hq8p): each its own owner.
	// Two waits on a person: the second move reads the board again after
	// the first wrote, which once hit a closed cache.
	for _, c := range [][2]string{{"Look", "nautilus"}, {"Build", "wyvern"}, {"Read", "kestrel"}} {
		ids[c[0]] = execute(t, a, "new", c[0], "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
		execute(t, a, "take", ids[c[0]], "--as", c[1], "--model", "m")
	}
	// As hug wrote them before waiting_for.
	execute(t, a, "wait", ids["Look"], "jhf: look at the colours", "--as", "nautilus")
	execute(t, a, "wait", ids["Build"], "CI run: 9", "--as", "wyvern")
	execute(t, a, "wait", ids["Read"], "jhf: read the note", "--as", "kestrel")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no")
	r := execute(t, a, "upgrade", "--apply")
	if did := strings.Join(r["upgrade"].([]string), "; "); !strings.Contains(did, "2 questions to a person moved from waiting to asked") {
		t.Fatalf("upgrade did: %s", did)
	}
	look, build := taskFile(t, a, ids["Look"]), taskFile(t, a, ids["Build"])
	if !strings.Contains(look, `asked_for: "jhf"`) || !strings.Contains(look, `asked_what: "look at the colours"`) || !strings.Contains(look, `asked_by: "nautilus"`) || !strings.Contains(look, `stage: "asked"`) || strings.Contains(look, "\nwaiting_") {
		t.Fatalf("Look not moved:\n%s", look)
	}
	if read := taskFile(t, a, ids["Read"]); !strings.Contains(read, `asked_for: "jhf"`) {
		t.Fatalf("Read not moved:\n%s", read)
	}
	if strings.Contains(build, "\nasked_") || !strings.Contains(build, `waiting_on: "CI run: 9"`) {
		t.Fatalf("Build changed:\n%s", build)
	}
	r = execute(t, a, "upgrade", "--apply")
	if did := strings.Join(r["upgrade"].([]string), "; "); strings.Contains(did, "asked") {
		t.Fatalf("second upgrade moved again: %s", did)
	}
}

// 41hhgn check 4: a wait as v0.7.50 wrote it, waiting_for and waiting_show
// with a "check N:" question (13 tasks on hug's own board), lints as the
// old layout and upgrades to asked with no waiting_ field left. Those two
// keys are no node field any more, so the upgrade reads them off the file.
func TestUpgradeMovesWaitingForFromFile(t *testing.T) {
	t.Parallel()
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--accept", "b")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "attach", id, "--name", "rows.ans", "--content", "rows")
	execute(t, a, "wait", id, "check 2: look at the rows", "--as", "nautilus")
	board := filepath.Join(a.Cwd, TaskDir)
	m, _ := filepath.Glob(filepath.Join(board, "*"+id+"*.md"))
	raw, _ := os.ReadFile(m[0])
	old := strings.Replace(string(raw), "\nwaiting_on:", "\nwaiting_for: \"jhf\"\nwaiting_show: \"rows.ans\"\nwaiting_on:", 1)
	if e := os.WriteFile(m[0], []byte(old), 0o644); e != nil {
		t.Fatal(e)
	}
	git(t, board, "-c", "core.hooksPath=/dev/null", "commit", "-qam", "as v0.7.50 wrote it")
	if code, out := runText(a, "lint", id); !strings.Contains(out, "old layout: waiting_for: is now a question, stage asked") || !strings.Contains(out, "hug upgrade") {
		t.Fatalf("lint of a v0.7.50 wait (exit %d):\n%s", code, out)
	}
	r := execute(t, a, "upgrade", "--apply")
	if did := strings.Join(r["upgrade"].([]string), "; "); !strings.Contains(did, "1 questions to a person moved from waiting to asked ("+id+")") {
		t.Fatalf("upgrade did: %s", did)
	}
	f := taskFile(t, a, id)
	for _, want := range []string{`stage: "asked"`, `asked_for: "jhf"`, `asked_check: "2"`, `asked_what: "look at the rows"`, `asked_show: "rows.ans"`, `asked_by: "nautilus"`} {
		if !strings.Contains(f, want) {
			t.Fatalf("upgraded file lacks %s:\n%s", want, f)
		}
	}
	if strings.Contains(f, "\nwaiting_") {
		t.Fatalf("a waiting_ field stayed:\n%s", f)
	}
	if _, out := runText(a, "lint", id); strings.Contains(out, "old-layout") || strings.Contains(out, "old layout") {
		t.Fatalf("lint after upgrade:\n%s", out)
	}
}

// 5z4zt4 (41hhgn: ask --show): hug task ask --show and design --show name the task file the
// person should look at; attach takes a PNG; --show of a file the task does
// not carry is refused with a fix.
func TestWaitShowsAPicture(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	dir := t.TempDir()
	png := filepath.Join(dir, "shot.png")
	if e := os.WriteFile(png, []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR"), 0o644); e != nil {
		t.Fatal(e)
	}
	ans := filepath.Join(dir, "rows.ans")
	if e := os.WriteFile(ans, []byte("\x1b[31mIN PROGRESS\x1b[0m\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	execute(t, a, "attach", id, png, ans)
	_, e := a.Execute(context.Background(), []string{"ask", id, "look", "--to", "jhf", "--show", "nope.ans"})
	if e == nil || !strings.Contains(e.Error(), "hug task attach "+id+" FILE --name nope.ans") {
		t.Fatalf("--show of a missing file: %v", e)
	}
	execute(t, a, "ask", id, "look at the rows", "--to", "jhf", "--show", "rows.ans")
	if f := taskFile(t, a, id); !strings.Contains(f, `asked_show: "rows.ans"`) || !strings.Contains(f, `asked_for: "jhf"`) {
		t.Fatalf("ask --show:\n%s", f)
	}
	// Asked again without --show: no picture.
	execute(t, a, "ask", id, "and now?", "--to", "jhf")
	if f := taskFile(t, a, id); strings.Contains(f, "\nasked_show:") {
		t.Fatalf("ask without --show kept asked_show:\n%s", f)
	}
	todo := execute(t, a, "new", "Ask", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "attach", todo, ans)
	execute(t, a, "design", todo, "this colour or that?", "--for", "jhf", "--show", "rows.ans")
	if f := taskFile(t, a, todo); !strings.Contains(f, `design_show: "rows.ans"`) {
		t.Fatalf("design --show:\n%s", f)
	}
	execute(t, a, "answer", todo, "this one", "--as", "jhf")
	if f := taskFile(t, a, todo); strings.Contains(f, "\ndesign_show:") {
		t.Fatalf("answer kept design_show:\n%s", f)
	}
}

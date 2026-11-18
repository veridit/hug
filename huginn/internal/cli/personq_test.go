package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// personBoard is a board with Statbus's [people]: jhf (Jørgen H. Fjeld),
// erik (Erik Søberg) and hhz (Hedda), and one doing task with six checks
// taken by rabbit.
func personBoard(t *testing.T) (*App, string) {
	t.Helper()
	a := app(t)
	execute(t, a, "init")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no", "--name", "Jørgen H. Fjeld")
	execute(t, a, "people", "add", "erik", "eriksoeb@gmail.com", "--name", "Erik Søberg")
	execute(t, a, "people", "add", "hhz", "hhz@ssb.no", "--name", "Hedda")
	args := []string{"new", "Release notes", "--goal", "g", "--purpose", "p", "--principle", "q"}
	for i := 1; i <= 6; i++ {
		args = append(args, "--accept", "check "+strconv.Itoa(i))
	}
	id := execute(t, a, args...)["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "rabbit", "--model", "m")
	return a, id
}

// agxqax check 1: a wait whose reason addresses a person ('owner: ...', a
// [people] name or alias, 'the owner's ...') is refused with the ask that
// says it, and writes nothing; a wait on a thing, or on an agent, is kept.
func TestWaitAddressedToPersonIsRefused(t *testing.T) {
	t.Parallel()
	a, id := personBoard(t)
	for _, c := range []struct{ on, fix string }{
		// Statbus hmgadt, as rabbit wrote it at 12:28.
		{"owner: check 4 on Norway rc.02 (remove a test record through the form)", `hug task ask ` + id + ` --to jhf --check 4 "check 4 on Norway rc.02 (remove a test record through the form)"`},
		{"owner: check 6 (plain paging, frame-grouped, or both)", `hug task ask ` + id + ` --to jhf --check 6 "check 6 (plain paging, frame-grouped, or both)"`},
		// Statbus j2g6wd, hibiscus at 00:40.
		{"the owner's decision on hug upgrade --apply (hug v0.7.53 is installed)", `hug task ask ` + id + ` --to jhf "the owner's decision on hug upgrade --apply (hug v0.7.53 is installed)"`},
		{"jhf: check 2: look at the rows", `hug task ask ` + id + ` --to jhf --check 2 "look at the rows"`},
		{"Jørgen: approve the colours", `hug task ask ` + id + ` --to jhf "approve the colours"`},
		{"jorgen: approve the colours", `hug task ask ` + id + ` --to jhf "approve the colours"`},
		{"Erik's go-ahead on the import", `hug task ask ` + id + ` --to erik "Erik's go-ahead on the import"`},
		{"waiting for the owner to merge PR 12", `hug task ask ` + id + ` --to jhf "waiting for the owner to merge PR 12"`},
	} {
		code, out := runText(a, "wait", id, c.on, "--as", "rabbit")
		if code != 2 || !strings.Contains(out, "a question to a person is hug task ask") || !strings.Contains(out, "fix: "+c.fix+"\n") {
			t.Fatalf("wait %q: exit %d, want 2 with fix %s:\n%s", c.on, code, c.fix, out)
		}
		if f := taskFile(t, a, id); strings.Contains(f, "waiting_on") || strings.Contains(f, `stage: "waiting"`) {
			t.Fatalf("a refused wait %q wrote:\n%s", c.on, f)
		}
	}
	// --until does not make a question a wait.
	if code, out := runText(a, "wait", id, "owner: check 3", "--until", "2099-01-01", "--as", "rabbit"); code != 2 || !strings.Contains(out, "hug task ask "+id+" --to jhf") {
		t.Fatalf("wait --until on the owner: exit %d\n%s", code, out)
	}
	// Things, and agents, are waited on.
	for _, on := range []string{"CI run 7", "release v0.7.64 exists", "crab: landing queue after qpp30y", "stale-owner lint lands (x1t1t1)", "PR approved by jhf, CI run 9"} {
		execute(t, a, "wait", id, on, "--as", "rabbit")
		if f := taskFile(t, a, id); !strings.Contains(f, `waiting_on: `+strconv.Quote(on)) {
			t.Fatalf("wait %q not written:\n%s", on, f)
		}
		execute(t, a, "resume", id)
	}
	// The fix runs: the question becomes stage asked, for jhf.
	runFix(t, a, `hug task ask `+id+` --to jhf --check 4 "check 4 on Norway rc.02 (remove a test record through the form)"`)
	if f := taskFile(t, a, id); !strings.Contains(f, `stage: "asked"`) || !strings.Contains(f, `asked_for: "jhf"`) || !strings.Contains(f, `asked_check: "4"`) {
		t.Fatalf("the fix did not ask:\n%s", f)
	}
}

// agxqax check 2: ask --to a person's long name or alias resolves to their
// short name; a name that is nobody here is refused naming the people.
func TestAskToResolvesOrRefusesName(t *testing.T) {
	t.Parallel()
	a, id := personBoard(t)
	for _, to := range []string{"jorgen", "Jørgen", "JHF", "Jørgen H. Fjeld"} {
		execute(t, a, "ask", id, "--to", to, "confirm time frames on Norway", "--as", "rabbit")
		if f := taskFile(t, a, id); !strings.Contains(f, `asked_for: "jhf"`) {
			t.Fatalf("ask --to %q did not resolve to jhf:\n%s", to, f)
		}
		execute(t, a, "resume", id)
	}
	// An alias in hug.toml resolves too, and people add keeps it.
	cfg := filepath.Join(a.Cwd, TaskDir, task.ConfigName)
	b, _ := os.ReadFile(cfg)
	if e := os.WriteFile(cfg, []byte(strings.Replace(string(b), "name = \"Hedda\"\n", "name = \"Hedda\"\nalias = [\"hedda-ssb\"]\n", 1)), 0o644); e != nil {
		t.Fatal(e)
	}
	execute(t, a, "people", "add", "hhz", "hedda@example.org")
	if b, _ = os.ReadFile(cfg); !strings.Contains(string(b), `alias = ["hedda-ssb"]`) {
		t.Fatalf("people add dropped the alias:\n%s", b)
	}
	execute(t, a, "ask", id, "--to", "hedda-ssb", "the colours?", "--as", "rabbit")
	if f := taskFile(t, a, id); !strings.Contains(f, `asked_for: "hhz"`) {
		t.Fatalf("alias did not resolve:\n%s", f)
	}
	execute(t, a, "resume", id)
	// Nobody: refused, naming who is here; nothing written.
	code, out := runText(a, "ask", id, "--to", "bob", "the colours?", "--as", "rabbit")
	if code != 2 || !strings.Contains(out, "bob") || !strings.Contains(out, "jhf (Jørgen H. Fjeld)") || !strings.Contains(out, "fix: hug task ask "+id+` --to jhf "the colours?"`) {
		t.Fatalf("ask --to bob: exit %d\n%s", code, out)
	}
	if f := taskFile(t, a, id); strings.Contains(f, "\nasked_for:") || !strings.Contains(f, `stage: "active"`) {
		t.Fatalf("a refused ask wrote:\n%s", f)
	}
	// An agent on the board, and the coordinator, are asked as written.
	execute(t, a, "ask", id, "--to", "rabbit", "is the diff right?", "--as", "rabbit")
	execute(t, a, "resume", id)
	execute(t, a, "coordinator", "crab")
	execute(t, a, "ask", id, "--to", "crab", "land it?", "--as", "rabbit")
	if f := taskFile(t, a, id); !strings.Contains(f, `asked_for: "crab"`) {
		t.Fatalf("ask the coordinator:\n%s", f)
	}
}

// agxqax check 3: lint flags a waiting task whose reason addresses a
// person, and an asked task whose asked_for is a person's other name, each
// with a fix that runs and leaves lint quiet. The tasks are written as
// Statbus's were, before hug refused them.
func TestLintFlagsPersonWaitsAndUnknownAsk(t *testing.T) {
	t.Parallel()
	a, wait := personBoard(t)
	asked := execute(t, a, "new", "Time frames", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", asked, "--as", "otter", "--model", "m")
	execute(t, a, "wait", wait, "CI run 1", "--as", "rabbit")
	execute(t, a, "ask", asked, "--to", "jhf", "owner confirms time frames on Norway after the release (rc.02 shipped)", "--as", "otter")
	board := filepath.Join(a.Cwd, TaskDir)
	rewrite := func(id, from, to string) {
		m, _ := globID(board, id)
		raw, _ := os.ReadFile(m[0])
		if !strings.Contains(string(raw), from) {
			t.Fatalf("%s lacks %q:\n%s", id, from, raw)
		}
		if e := os.WriteFile(m[0], []byte(strings.Replace(string(raw), from, to, 1)), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	rewrite(wait, `waiting_on: "CI run 1"`, `waiting_on: "owner: check 4 on Norway rc.02 (remove a test record through the form)"`)
	rewrite(asked, `asked_for: "jhf"`, `asked_for: "jorgen"`)
	git(t, board, "-c", "core.hooksPath=/dev/null", "commit", "-qam", "as Statbus wrote them")

	_, out := runText(a, "lint")
	for _, want := range []string{"wait-on-person", "unknown-person"} {
		if !strings.Contains(out, want) {
			t.Fatalf("lint lacks %s:\n%s", want, out)
		}
	}
	waitFix := `hug task ask ` + wait + ` --to jhf --check 4 "check 4 on Norway rc.02 (remove a test record through the form)"`
	askFix := `hug task ask ` + asked + ` --to jhf "owner confirms time frames on Norway after the release (rc.02 shipped)"`
	for id, fix := range map[string]string{wait: waitFix, asked: askFix} {
		_, out = runText(a, "lint", id)
		if !strings.Contains(out, "next: "+fix+"\n") && !strings.Contains(out, "fix: "+fix+"\n") {
			t.Fatalf("lint %s lacks the fix %s:\n%s", id, fix, out)
		}
		runFix(t, a, fix)
	}
	if _, out = runText(a, "lint"); !strings.HasPrefix(out, "no findings") {
		t.Fatalf("lint after the fixes:\n%s", out)
	}
	if f := taskFile(t, a, asked); !strings.Contains(f, `asked_for: "jhf"`) {
		t.Fatalf("asked task not re-asked of jhf:\n%s", f)
	}
}

// runFix runs a fix line as the agent would: hug task CMD args, with
// double-quoted arguments, as the owner of the task.
func runFix(t *testing.T, a *App, fix string) {
	t.Helper()
	rest, ok := strings.CutPrefix(fix, "hug task ")
	if !ok {
		t.Fatalf("not a hug task command: %s", fix)
	}
	var args []string
	for rest = strings.TrimSpace(rest); rest != ""; rest = strings.TrimSpace(rest) {
		if rest[0] == '"' {
			q, e := strconv.QuotedPrefix(rest)
			if e != nil {
				t.Fatalf("fix %s: %v", fix, e)
			}
			v, _ := strconv.Unquote(q)
			args, rest = append(args, v), rest[len(q):]
			continue
		}
		w, r, _ := strings.Cut(rest, " ")
		args, rest = append(args, w), r
	}
	if code, out := runText(a, args...); code != 0 {
		t.Fatalf("fix %s: exit %d\n%s", fix, code, out)
	}
}

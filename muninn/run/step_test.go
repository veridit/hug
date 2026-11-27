package run

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/veridit/hug/huginn/board"
	huginn "github.com/veridit/hug/huginn/run"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/store"
)

// erv680: hug names the step due now, for the role it is due for, with one
// command, the same in the CLI's next: line, the MCP result's next and the
// board cue. One board, one task, its whole life: taken with no checks,
// checks written, ticked, sent to review, approved, shipped (the project's
// way: hug.toml [board] ship, else its AGENTS.md), done.
type stepWorld struct {
	t    *testing.T
	repo string
	cs   *sdk.ClientSession
}

func newStepWorld(t *testing.T) *stepWorld {
	t.Helper()
	memHome(t)
	for k, v := range map[string]string{"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@e", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@e", "HUG_USER": "", "HUG_MODEL": "m", "JCODE_SESSION_ID": "", "HUG_SYNC": "off"} {
		t.Setenv(k, v)
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	exec.Command("git", "-C", repo, "commit", "-q", "--allow-empty", "-m", "init").Run()
	t.Chdir(repo)
	w := &stepWorld{t: t, repo: repo}
	w.hug("init")
	ctx := context.Background()
	s := sdk.NewServer(&sdk.Implementation{Name: "hug", Version: "t"}, nil)
	if err := huginn.RegisterMCP(s, "task_"); err != nil {
		t.Fatal(err)
	}
	st, ct := sdk.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ss.Close() })
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cs.Close() })
	w.cs = cs
	return w
}

// hug runs hug task args and returns its text output.
func (w *stepWorld) hug(args ...string) string {
	w.t.Helper()
	out, _ := captureSurface(w.t, func() error {
		if code := huginn.Main(args); code != 0 {
			w.t.Fatalf("hug task %v: exit %d", args, code)
		}
		return nil
	})
	return out
}

var nextLine = regexp.MustCompile(`(?m)^next: (.*)$`)

// as runs hug task args as who (HUG_USER: show and set take no --as).
func (w *stepWorld) as(who string, args ...string) string {
	w.t.Helper()
	w.t.Setenv("HUG_USER", who)
	defer w.t.Setenv("HUG_USER", "")
	return w.hug(args...)
}

// commit commits what a test wrote on the board by hand.
func (w *stepWorld) commit() {
	w.t.Helper()
	dir := filepath.Join(w.repo, ".hug", "tasks")
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "by hand"}} {
		if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			w.t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
}

// next is the CLI's next: line of hug task args, as who.
func (w *stepWorld) next(who string, args ...string) string {
	w.t.Helper()
	out := w.as(who, args...)
	m := nextLine.FindAllStringSubmatch(out, -1)
	if len(m) != 1 {
		w.t.Fatalf("hug task %v: want one next: line, got %d:\n%s", args, len(m), out)
	}
	return m[0][1]
}

// mcpNext is the MCP task_show result's next for id, as who.
func (w *stepWorld) mcpNext(id, who string) (string, map[string]any) {
	w.t.Helper()
	w.t.Setenv("HUG_USER", who)
	defer w.t.Setenv("HUG_USER", "")
	out, err := w.cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "task_show", Arguments: map[string]any{"id": id}})
	if err != nil {
		w.t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out.Content[0].(*sdk.TextContent).Text), &m); err != nil {
		w.t.Fatal(err)
	}
	n, _ := m["next"].(string)
	return n, m
}

// cue is the board line session who hears now (a fresh engine: no rate).
func (w *stepWorld) cue(who string) (line, kind string) {
	w.t.Helper()
	s, err := board.Open(context.Background(), w.repo)
	if err != nil || s == nil {
		w.t.Fatalf("open: %v %v", s, err)
	}
	defer s.Close()
	db, err := store.Open(filepath.Join(w.t.TempDir(), "m.db"))
	if err != nil {
		w.t.Fatal(err)
	}
	defer db.Close()
	e := &recall.Engine{DB: db}
	in := recall.BoardInput{Tasks: boardTasksFor(s.Tasks, who), Coordinator: s.Coordinator, Self: recall.BoardSelf{Name: who}}
	return e.BoardCue("s-"+who, "go on", "", time.Now(), func(time.Time, bool) *recall.BoardInput { return &in })
}

// commands counts the hug commands a line names.
func commands(line string) int { return strings.Count(line, "hug task ") }

// TestStepOverATasksLife: each step names exactly one next command, and
// the CLI's next:, MCP's next and the board cue (for the role it is due
// for) name the same one (erv680 check 3). Taking a task with no
// acceptance checks names writing them (check 1); after approval the cue
// names the stage, ship it, and the project's way (check 5, both ways).
func TestStepOverATasksLife(t *testing.T) {
	for _, shipLine := range []string{"", "merge the branch to main"} {
		name := "ship line unset"
		if shipLine != "" {
			name = "ship line set"
		}
		t.Run(name, func(t *testing.T) {
			w := newStepWorld(t)
			w.hug("coordinator", "crab")
			if shipLine != "" {
				toml := filepath.Join(w.repo, ".hug", "tasks", "hug.toml")
				b, _ := os.ReadFile(toml)
				if !strings.Contains(string(b), "[board]\n") {
					t.Fatalf("hug.toml has no [board]:\n%s", b)
				}
				os.WriteFile(toml, []byte(strings.Replace(string(b), "[board]\n", "[board]\nship = \""+shipLine+"\"\n", 1)), 0o644)
				w.commit()
			}
			id := regexp.MustCompile(`"id":\s*"([a-z0-9]{6})"`).FindStringSubmatch(w.hug("new", "Self guiding", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "placeholder", "--json"))[1]
			// A task with no acceptance checks: written by hand, as an
			// import or the TUI's one-line add can leave it.
			path := filepath.Join(w.repo, ".hug", "tasks", id[:1], id+".md")
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b), "- [ ] placeholder\n") {
				t.Fatalf("no placeholder check:\n%s", b)
			}
			os.WriteFile(path, []byte(strings.Replace(string(b), "- [ ] placeholder\n", "", 1)), 0o644)
			w.commit()

			// same checks that CLI, MCP and the cue name cmd for who.
			same := func(step, who, cliNext, cmd string) {
				t.Helper()
				if cliNext != cmd || commands(cliNext) != 1 {
					t.Fatalf("%s: CLI next: %q, want %q (one command)", step, cliNext, cmd)
				}
				if m, _ := w.mcpNext(id, who); m != cmd {
					t.Fatalf("%s: MCP next %q, want %q", step, m, cmd)
				}
				line, _ := w.cue(who)
				if !strings.HasSuffix(line, cmd) || commands(line) != 1 {
					t.Fatalf("%s: %s's cue %q, want it to end in %q, naming one command", step, who, line, cmd)
				}
			}

			// 1. Taken with no checks: write them (check 1).
			out := w.as("wolf", "take", id, "--model", "m")
			checksCmd := "hug task set " + id + ` --accept "observable check"`
			if !strings.Contains(out, "now: "+id+" has no acceptance checks: write them first") || !strings.Contains(out, "next: "+checksCmd+"\n") {
				t.Fatalf("take with no checks:\n%s", out)
			}
			same("no checks", "wolf", w.next("wolf", "show", id), checksCmd)
			if line, kind := w.cue("wolf"); kind != "checks" || !strings.Contains(line, "has no acceptance checks") {
				t.Fatalf("no-checks cue: %q %q", kind, line)
			}

			// 2. Checks written: tick the first.
			cli := w.next("wolf", "set", id, "--accept", "one holds", "--accept", "two holds")
			if cli != "hug task check "+id+" 1" {
				t.Fatalf("after writing checks: next %q", cli)
			}
			if m, _ := w.mcpNext(id, "wolf"); m != cli {
				t.Fatalf("after writing checks: MCP %q, CLI %q", m, cli)
			}
			if line, kind := w.cue("wolf"); kind == "checks" || strings.Contains(line, "has no acceptance checks") {
				t.Fatalf("checks written, the cue still asks for them: %q", line)
			}
			cli = w.next("wolf", "check", id, "1", "evidence: test one")
			if cli != "hug task check "+id+" 2" {
				t.Fatalf("after check 1: next %q", cli)
			}

			// 3. Every check holds: close it (5yh7hr makes this the
			// review step later).
			cli = w.next("wolf", "check", id, "2", "evidence: test two")
			same("every check holds", "wolf", cli, "hug task done "+id)

			// 4. In review: the reviewer's step, not the owner's.
			cli = w.next("wolf", "review", id, "--by", "crab")
			if cli != "hug task next" {
				t.Fatalf("owner after asking for review: next %q, want hug task next", cli)
			}
			review := "hug task answer " + id + ` "looks right"`
			same("in review", "crab", w.next("crab", "show", id), review)
			if line, kind := w.cue("crab"); kind != "review" || !strings.HasPrefix(line, "Task board: wolf asks you to review "+id) {
				t.Fatalf("reviewer's cue: %q %q", kind, line)
			}
			if line, _ := w.cue("wolf"); strings.Contains(line, review) {
				t.Fatalf("the owner is told to answer its own review: %q", line)
			}

			// 5. Approved: ship it, the project's way (check 5).
			how := "as your AGENTS.md says"
			if shipLine != "" {
				how = "as hug.toml says: " + shipLine
			}
			ship := "ship it, " + how + "; once it has shipped: hug task done " + id
			cli = w.next("crab", "answer", id, "looks right")
			same("approved", "crab", cli, ship)
			line, kind := w.cue("crab")
			if kind != "ship" || line != "Task board: crab approved "+id+" (Self guiding): "+ship {
				t.Fatalf("coordinator's cue after approval: %q %q", kind, line)
			}
			for _, word := range []string{"tag", "release", "tested SHA", "land"} {
				if strings.Contains(strings.ToLower(line), strings.ToLower(word)) {
					t.Fatalf("the ship cue names a procedure (%q): %q", word, line)
				}
			}
			if line, _ := w.cue("wolf"); strings.Contains(line, "ship it") || strings.Contains(line, "Close it") {
				t.Fatalf("the owner is told the coordinator's step: %q", line)
			}

			// 6. Shipped and closed: the next task, and no cue on it.
			if cli = w.next("crab", "done", id); cli != "hug task next" {
				t.Fatalf("done: next %q", cli)
			}
			if m, _ := w.mcpNext(id, "crab"); m != "hug task next" {
				t.Fatalf("done: MCP next %q", m)
			}
			// The owner still hears the reviewer's answer once (3jtfw9):
			// news, not a step.
			for _, who := range []string{"crab", "wolf"} {
				if line, kind := w.cue(who); strings.Contains(line, id) && kind != "answer" {
					t.Fatalf("done, %s still hears a step on it: %q %q", who, kind, line)
				}
			}
		})
	}
}

// A review answered with what to change is no approval: the owner works
// on, and the cue names no ship step (erv680 check 5).
func TestStepChangesAskedIsNoApproval(t *testing.T) {
	w := newStepWorld(t)
	w.hug("coordinator", "crab")
	id := regexp.MustCompile(`"id":\s*"([a-z0-9]{6})"`).FindStringSubmatch(w.hug("new", "Self guiding", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a holds", "--json"))[1]
	w.as("wolf", "take", id, "--model", "m")
	w.as("wolf", "check", id, "1", "evidence: test a")
	w.as("wolf", "review", id, "--by", "crab")
	if cli := w.next("crab", "answer", id, "rename the flag, then ask again"); cli != "hug task done "+id {
		t.Fatalf("changes asked: next %q", cli)
	}
	if line, kind := w.cue("crab"); kind == "ship" || strings.Contains(line, "ship it") {
		t.Fatalf("changes asked, the coordinator hears ship it: %q", line)
	}
}

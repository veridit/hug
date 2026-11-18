package cli

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// ec6z6j check 2 (CLI half): hug task request ID1,ID2 "text" (or a
// repeatable --task) writes one request for the exact selection: the text
// once on the first task, a link on each other, one commit, and one
// delivery to the coordinator listing every target with its own resolve
// line. show and requests list each target pending or resolved; the
// request closes only when the last one resolves (check 3).
func TestRequestForSeveralTasksCLI(t *testing.T) {
	t.Parallel()
	jc := newFakeJcode(t)
	crab := jc.session(t, "crab", 2000, true)
	a := app(t)
	a.JcodeHomes = []string{jc.home}
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	var ids []string
	for _, title := range []string{"Tags", "Rows", "Header"} {
		ids = append(ids, execute(t, a, "new", title, "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string))
	}
	x, y, z := ids[0], ids[1], ids[2]
	r := execute(t, a, "request", x+","+y, "same fix on each:\nper board", "--task", z, "--as", "jhf")
	if r["receipt"] != "pending, delivered to crab ("+crab+")" {
		t.Fatalf("receipt: %v", r["receipt"])
	}
	msgs := jc.messages()
	if len(msgs) != 1 {
		t.Fatalf("delivered %d times, want once: %q", len(msgs), msgs)
	}
	_, msg, _ := strings.Cut(msgs[0], "\x00")
	board := realPath(t, a.Cwd)
	for _, want := range []string{
		"hug: jhf wrote request 1 on " + x + " (Tags) for 3 tasks: " + x + ", " + y + ", " + z + " on board " + filepath.Base(board) + " (" + board + "), to you as the board coordinator:\n",
		"  same fix on each:\n  per board\n",
		"Turn it into changes to each ticket, and resolve each one (in " + board + "):\n",
		"  hug task resolve " + x + " 1 \"what changed\"\n",
		"  hug task resolve " + y + " 1 \"what changed\"\n",
		"  hug task resolve " + z + " 1 \"what changed\"\n",
		"[hug " + x + " request 1 ",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message lacks %q:\n%s", want, msg)
		}
	}
	// One commit for the request, then one for its receipt.
	if out := git(t, filepath.Join(a.Cwd, TaskDir), "log", "--format=%s", "-2"); out != "hug task request 1 on "+x+" delivered to crab\nhug task request "+x+","+y+","+z {
		t.Fatalf("commits:\n%s", out)
	}
	_, out := runText(a, "show", y)
	if !strings.Contains(out, "request 1 by jhf, ") || !strings.Contains(out, "(open): see "+x+" request 1:\n") || !strings.Contains(out, "  same fix on each:\n  per board\n") {
		t.Fatalf("show on a link lacks the request:\n%s", out)
	}
	_, out = runText(a, "show", x)
	if !strings.Contains(out, "  targets: "+x+" pending, "+y+" pending, "+z+" pending\n") {
		t.Fatalf("show on the home lacks the targets:\n%s", out)
	}
	execute(t, a, "resolve", y, "1", "fixed on Rows", "--as", "crab")
	execute(t, a, "resolve", x, "1", "fixed on Tags", "--as", "crab")
	_, out = runText(a, "requests", x)
	if !strings.Contains(out, "  targets: "+x+" resolved, "+y+" resolved, "+z+" pending\n") || strings.Contains(out, "receipt: answered") {
		t.Fatalf("requests after two of three:\n%s", out)
	}
	_, out = runText(a, "requests")
	if !strings.Contains(out, z+" request 1 by jhf") || strings.Contains(out, y+" request 1 by jhf") {
		t.Fatalf("open requests: the pending link must show, the resolved one not:\n%s", out)
	}
	execute(t, a, "resolve", z, "1", "fixed on Header", "--as", "crab")
	_, out = runText(a, "requests", x, "--all")
	if !strings.Contains(out, "  targets: "+x+" resolved, "+y+" resolved, "+z+" resolved\n") || !strings.Contains(out, "  receipt: answered\n") {
		t.Fatalf("requests after the last:\n%s", out)
	}
	if f := taskFile(t, a, x); !strings.Contains(f, " request 1 closed: all targets resolved\n") {
		t.Fatalf("home History:\n%s", f)
	}
}

// ec6z6j (ruling 3): an unknown ID in the selection writes nothing,
// exit 2, with the request on the known ones as the fix.
func TestRequestForUnknownCLI(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	x := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	before := git(t, filepath.Join(a.Cwd, TaskDir), "rev-parse", "HEAD")
	code, out := runText(a, "request", x+",zzzzzz", "fix it", "--as", "jhf")
	if code != 2 || !strings.Contains(out, "no task zzzzzz: nothing was written") || !strings.Contains(out, "fix: hug task request "+x+` "fix it"`) {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if after := git(t, filepath.Join(a.Cwd, TaskDir), "rev-parse", "HEAD"); after != before {
		t.Fatalf("a refused request committed: %s -> %s", before, after)
	}
	if _, e := a.Execute(context.Background(), []string{"request", "--task", x, "--as", "jhf"}); e == nil || !strings.Contains(e.Error(), "request needs the request") {
		t.Fatalf("--task without text: %v", e)
	}
}

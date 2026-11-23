package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// z2vmte through the commands: wait --until and ask --after park the task;
// before the date ls changes nothing; on the date the first read (ls)
// wakes it in one commit, and next --mine leads with it. Not parallel: it
// sets the clock.
func TestTimedWaitCommands(t *testing.T) {
	old := task.Now
	t.Cleanup(func() { task.Now = old })
	day := func(d int) { task.Now = func() time.Time { return time.Date(2026, 11, d, 9, 0, 0, 0, time.Local) } }
	day(18)
	a := app(t)
	execute(t, a, "init")
	newTask := func(title string) string {
		return execute(t, a, "new", title, "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	}
	w, q := newTask("Look again"), newTask("Ask later")
	execute(t, a, "take", q, "--as", "crab", "--model", "m")
	execute(t, a, "ask", q, "--to", "kestrel", "--after", "2026-11-24", "--check", "1", "has he had to remind an agent?", "--as", "crab")
	execute(t, a, "take", w, "--as", "kestrel", "--model", "m")
	execute(t, a, "wait", w, "--until", "2026-11-24", "a week of use", "--as", "kestrel")
	if f := taskFile(t, a, w); !strings.Contains(f, `waiting_until: "2026-11-24"`) {
		t.Fatalf("wait --until:\n%s", f)
	}
	if _, out := runText(a, "show", w); !strings.Contains(out, "stage: waiting until 2026-11-24 · on a week of use") { // atex3d: the date first
		t.Fatalf("show before the date:\n%s", out)
	}
	if f := taskFile(t, a, q); strings.Contains(f, "\nasked_") {
		t.Fatalf("ask --after asked now:\n%s", f)
	}
	board := a.Cwd + "/" + TaskDir
	head := git(t, board, "rev-parse", "HEAD")
	day(23)
	execute(t, a, "ls")
	if now := git(t, board, "rev-parse", "HEAD"); now != head {
		t.Fatalf("ls on 23 Nov committed: %s", git(t, board, "log", "--oneline", "-3"))
	}
	day(24)
	execute(t, a, "ls")
	log := git(t, board, "log", "--format=%s", head+"..HEAD")
	if strings.Count(log, "\n") != 0 || !strings.Contains(log, "hug task wake: ") || !strings.Contains(log, w) || !strings.Contains(log, q) {
		t.Fatalf("one wake commit for both, got:\n%s", log)
	}
	front := func(f string) string { return strings.SplitN(f, "\n---\n", 2)[0] }
	if f := taskFile(t, a, w); !strings.Contains(f, `stage: "active"`) || strings.Contains(front(f), "waiting_") {
		t.Fatalf("woken wait:\n%s", f)
	}
	if f := taskFile(t, a, q); !strings.Contains(f, `stage: "asked"`) || !strings.Contains(f, `asked_what: "has he had to remind an agent?"`) || !strings.Contains(f, `asked_check: "1"`) || !strings.Contains(f, `asked_for: "kestrel"`) {
		t.Fatalf("woken ask:\n%s", f)
	}
	r := execute(t, a, "next", "--mine", "--as", "kestrel")
	if got := r["task"].(Reply)["id"]; got != w {
		t.Fatalf("next --mine: %v, want %s", got, w)
	}
	// A second read the same day writes nothing.
	head = git(t, board, "rev-parse", "HEAD")
	execute(t, a, "ls")
	if now := git(t, board, "rev-parse", "HEAD"); now != head {
		t.Fatalf("second read committed")
	}
}

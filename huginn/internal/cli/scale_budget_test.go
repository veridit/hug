package cli

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestScaleBudget10K builds a board of 10,000 tasks (the shape of
// gen-scale-board.py, attached to 44w1ev: groups, epics, dependencies,
// notes) and fails when ls, next, show, board, lint or note exceed the
// budget once the cache is warm. The target is 400 ms on a developer
// machine (Jørgen, 2026-11-16); the budget here is 3x that, for shared CI
// runners and the race detector, so it catches a return to the old cost
// (1.4-3.4 s at 10,000 tasks) without flaking. Skipped with -short.
//
// Times are measured in-process: what the command itself costs, without
// process start (~25 ms).
func TestScaleBudget10K(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test: 10,000 task files")
	}
	const n = 10000
	budget := 1200 * time.Millisecond
	if raceEnabled {
		budget *= 4 // the race detector slows everything 4-10x
	}
	a := app(t)
	execute(t, a, "init")
	root := filepath.Join(a.Cwd, ".hug", "tasks")
	writeScaleBoard(t, root, n)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "scale board")

	run := func(args ...string) time.Duration {
		t.Helper()
		a.Out = &bytes.Buffer{}
		t0 := time.Now()
		if code := a.Run(context.Background(), args); code != 0 && code != 4 {
			t.Fatalf("%v: exit %d: %s", args, code, a.Out)
		}
		return time.Since(t0)
	}
	t0 := time.Now()
	run("ls") // cold: parses every file into the cache
	t.Logf("cold ls (parse %d files): %v", n, time.Since(t0).Round(time.Millisecond))
	ls := execute(t, a, "ls")["tasks"].([]Reply)
	if len(ls) != n {
		t.Fatalf("ls lists %d tasks, want %d", len(ls), n)
	}
	id := ls[n/2]["id"].(string)
	for _, c := range [][]string{{"ls"}, {"next"}, {"show", id}, {"board"}, {"lint"}, {"note", id, "scale note"}} {
		var ts []time.Duration
		for i := 0; i < 3; i++ {
			ts = append(ts, run(c...))
		}
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		med := ts[1]
		t.Logf("%-5s median %v (target 400ms, budget %v)", c[0], med.Round(time.Millisecond), budget)
		if med > budget {
			t.Errorf("%s at %d tasks: median %v over the %v budget", c[0], n, med.Round(time.Millisecond), budget)
		}
	}
}

// writeScaleBoard writes n synthetic task files like gen-scale-board.py:
// deterministic, 150 epics, 30% with a parent, 8% with a dependency, a
// third in groups, notes of a few hundred words.
func writeScaleBoard(t testing.TB, root string, n int) {
	t.Helper()
	r := rand.New(rand.NewSource(7))
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"
	words := strings.Fields("retry parser cache login export import schema index worker queue report billing search upload token session layout router render config metric alert backup migrate sync")
	groups := []string{"", "", "", "api", "web", "compiler", "later", "ops"}
	statuses := []string{"todo", "todo", "todo", "doing", "done", "done", "dropped"}
	text := func(k int) string {
		w := make([]string, k)
		for i := range w {
			w[i] = words[r.Intn(len(words))]
		}
		return strings.Join(w, " ")
	}
	seen := map[string]bool{}
	ids := make([]string, 0, n)
	for len(ids) < n {
		b := make([]byte, 6)
		for i := range b {
			b[i] = alphabet[r.Intn(len(alphabet))]
		}
		if !seen[string(b)] {
			seen[string(b)] = true
			ids = append(ids, string(b))
		}
	}
	for i, id := range ids {
		st := statuses[r.Intn(len(statuses))]
		box := " "
		if st == "done" {
			box = "x"
		}
		parent, deps := "", ""
		if i >= 150 && r.Intn(10) < 3 {
			parent = "parent: " + ids[r.Intn(150)] + "\n"
		}
		if i > 200 && r.Intn(100) < 8 {
			deps = "- task:" + ids[150+r.Intn(i-150)] + "\n"
		}
		assignee := "[]"
		if st == "doing" {
			assignee = `["tester"]`
		}
		var checks strings.Builder
		for c := 0; c < 1+r.Intn(5); c++ {
			fmt.Fprintf(&checks, "- [%s] %s\n", box, text(4+r.Intn(10)))
		}
		var notes []string
		for c := 0; c < r.Intn(5); c++ {
			notes = append(notes, text(20+r.Intn(100)))
		}
		body := fmt.Sprintf("---\nid: %s\ntitle: \"%s\"\nstatus: %s\ntype: feature\npriority: \"medium\"\nassignee: %s\nlabels: []\n%screated: 2026-10-%02d\nupdated: 2026-11-%02d\n---\n\n## Goal\n%s\n\n## Purpose\n%s\n\n## Principle\n%s\n\n## Relevant details\n%s\n\n## Depends on\n%s\n## Acceptance\n%s\n## Notes\n%s\n",
			id, text(3+r.Intn(7)), st, assignee, parent, 1+r.Intn(28), 1+r.Intn(16), text(18), text(14), text(14), text(40), deps, checks.String(), strings.Join(notes, "\n\n"))
		dir := filepath.Join(root, groups[r.Intn(len(groups))])
		if e := os.MkdirAll(dir, 0o755); e != nil {
			t.Fatal(e)
		}
		name := fmt.Sprintf("%06d.%s.task-%d.md", i+1, id, i)
		if e := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
	}
}

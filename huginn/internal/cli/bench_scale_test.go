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

	"github.com/veridit/hug/huginn/internal/cache"
)

// TestScaleBench10K builds a board of 10,000 tasks (the shape of
// gen-scale-board.py, attached to 44w1ev: groups, epics, dependencies,
// notes) and fails when a warm command costs more than half of the same
// command on the uncached path: cache.db removed first, so it reads and
// parses every task file, the cost that grows with the board.
//
// A ratio, not an absolute budget (Jørgen, 2026-11-17 10:00): the same
// code measured 2-10x apart on two ubuntu runners (CI runs 44054586997 and
// 44057678727: ls 4.93 s over a 4.8 s race budget), so any fixed number
// flakes. Old and new run interleaved in one process, so a slow runner
// slows both. A benchmark, not a test: it runs only with HUG_BENCH=1 (the
// ci.yml bench job: ubuntu, no -race). The developer target stays 400 ms
// per command (Jørgen, 2026-11-16); the log prints both numbers.
func TestScaleBench10K(t *testing.T) {
	if os.Getenv("HUG_BENCH") != "1" {
		t.Skip("benchmark: HUG_BENCH=1 go test -run TestScaleBench10K ./huginn/internal/cli/")
	}
	const n, runs, maxRatio = 10000, 5, 0.5
	a := app(t)
	execute(t, a, "init")
	root := filepath.Join(a.Cwd, ".hug", "tasks")
	writeScaleBoard(t, root, n)
	git(t, root, "add", "-A")
	git(t, root, "commit", "-q", "-m", "scale board")
	db := cache.File(filepath.Join(a.Cwd, ".git"))

	run := func(args ...string) time.Duration {
		t.Helper()
		a.Out = &bytes.Buffer{}
		t0 := time.Now()
		if code := a.Run(context.Background(), args); code != 0 && code != 4 {
			t.Fatalf("%v: exit %d: %s", args, code, a.Out)
		}
		return time.Since(t0)
	}
	uncached := func(args ...string) time.Duration {
		t.Helper()
		for _, suffix := range []string{"", "-wal", "-shm"} {
			if e := os.Remove(db + suffix); e != nil && !os.IsNotExist(e) {
				t.Fatal(e)
			}
		}
		return run(args...)
	}
	uncached("ls")
	if _, e := os.Stat(db); e != nil {
		t.Fatalf("no cache at %s after ls: the uncached path measures nothing: %v", db, e)
	}
	ls := execute(t, a, "ls")["tasks"].([]Reply)
	if len(ls) != n {
		t.Fatalf("ls lists %d tasks, want %d", len(ls), n)
	}
	id := ls[n/2]["id"].(string)
	median := func(ts []time.Duration) time.Duration {
		sort.Slice(ts, func(i, j int) bool { return ts[i] < ts[j] })
		return ts[len(ts)/2]
	}
	// search (49n9kw): a word in about a tenth of the notes and a title word.
	for _, c := range [][]string{{"ls"}, {"next"}, {"show", id}, {"board"}, {"lint"}, {"search", "retry", "parser"}, {"note", id, "scale note"}} {
		var olds, news []time.Duration
		for i := 0; i < runs; i++ {
			olds = append(olds, uncached(c...))
			run(c...) // the uncached run left a fresh cache; settle it
			news = append(news, run(c...))
		}
		old, cur := median(olds), median(news)
		ratio := float64(cur) / float64(old)
		t.Logf("%-5s warm %v, uncached %v, ratio %.2f (limit %.2f; target 400ms warm)", c[0], cur.Round(time.Millisecond), old.Round(time.Millisecond), ratio, maxRatio)
		if ratio > maxRatio {
			t.Errorf("%s at %d tasks: warm %v is %.2f of the uncached %v, over %.2f: a command again does work that grows with the board", c[0], n, cur.Round(time.Millisecond), ratio, old.Round(time.Millisecond), maxRatio)
		}
	}
	// The search query itself (49n9kw check 5: under 100 ms at 10,000 tasks),
	// without process start or opening the board: in-process, on the warm
	// cache, against warm show, which opens the board just the same. A word
	// in about every task is the worst case for ranking.
	s, e := a.openBoard(context.Background(), false)
	if e != nil {
		t.Fatal(e)
	}
	defer s.c.Close()
	var qs, shows []time.Duration
	for i := 0; i < runs; i++ {
		t0 := time.Now()
		if _, e := s.c.Search(context.Background(), "retry", searchLimit, nil); e != nil {
			t.Fatal(e)
		}
		qs = append(qs, time.Since(t0))
		shows = append(shows, run("show", id))
	}
	q, show := median(qs), median(shows)
	t.Logf("search query warm %v (show %v, ratio %.2f, limit %.2f; target 100ms)", q.Round(time.Millisecond), show.Round(time.Millisecond), float64(q)/float64(show), maxSearchRatio)
	if float64(q) > maxSearchRatio*float64(show) {
		t.Errorf("search query at %d tasks: %v is %.2f of warm show %v, over %.2f: ranking reads more than it ranks", n, q.Round(time.Millisecond), float64(q)/float64(show), show.Round(time.Millisecond), maxSearchRatio)
	}
}

// maxSearchRatio bounds the in-process search query against warm show.
// Measured on board10k: query 31-39 ms, show 0.2 s, so about 0.2; reading
// every hit's texts (the first version) made it 150-205 ms, about 0.8.
const maxSearchRatio = 0.5

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
		body := fmt.Sprintf("---\nid: %s\ntitle: \"%s\"\nstate: %s\ntype: feature\npriority: \"medium\"\nassignee: %s\nlabels: []\n%screated: 2026-10-%02d\nupdated: 2026-11-%02d\n---\n\n## Goal\n%s\n\n## Purpose\n%s\n\n## Principle\n%s\n\n## Relevant details\n%s\n\n## Depends on\n%s\n## Acceptance\n%s\n## Notes\n%s\n",
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

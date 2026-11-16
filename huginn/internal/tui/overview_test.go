package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// placeBoard: Alpha todo, Bravo doing waiting on Delta, Charlie todo waiting
// on Delta, Delta todo, Echo done, Foxtrot dropped, Golf doing.
func placeBoard(t *testing.T) (model, map[string]string) {
	t.Helper()
	s := &task.Store{Root: t.TempDir()}
	ids := map[string]string{}
	for _, title := range []string{"Delta", "Alpha", "Bravo", "Charlie", "Echo", "Foxtrot", "Golf"} {
		o := task.NewOptions{Title: title, Goal: "G", Purpose: "P", Principle: "Q", Acceptance: []string{"C"}}
		if title == "Bravo" || title == "Charlie" {
			o.Deps = []string{ids["Delta"]}
		}
		r, err := s.New(o)
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
	}
	setStatus := func(title, status, owner string) {
		d, err := s.Get(ids[title])
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(s.Root, d.Task.Path)
		b, _ := os.ReadFile(path)
		text := strings.Replace(string(b), "status: todo", "status: "+status, 1)
		if owner != "" {
			text = strings.Replace(text, "assignee: []", `assignee: ["`+owner+`"]`, 1)
		}
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	setStatus("Bravo", "doing", "ann")
	setStatus("Golf", "doing", "bob")
	setStatus("Echo", "done", "")
	setStatus("Foxtrot", "dropped", "")
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 140, 40
	m.resize()
	return m, ids
}

func sectionIDs(m model, p int) []string {
	var out []string
	for _, i := range m.paneIndices(p) {
		out = append(out, m.visible[i].ID)
	}
	return out
}

// Check 1: a doing task with an unmet dependency is in IN PROGRESS only,
// drawn blocked; BLOCKED holds only todo tasks that cannot start; every
// open task is in exactly one of IN PROGRESS, NEXT UP and BLOCKED.
func TestOverviewOneSectionPerTask(t *testing.T) {
	m, ids := placeBoard(t)
	m.setView("overview")
	bravo := ids["Bravo"]
	doing, next, blocked := sectionIDs(m, sectionDoing), sectionIDs(m, sectionNext), sectionIDs(m, sectionBlocked)
	if !contains(doing, bravo) || contains(blocked, bravo) || contains(next, bravo) {
		t.Fatalf("Bravo (doing, unmet dependency): IN PROGRESS %v, NEXT UP %v, BLOCKED %v", doing, next, blocked)
	}
	if strings.Join(blocked, " ") != ids["Charlie"] {
		t.Fatalf("BLOCKED holds %v, want only Charlie %s", blocked, ids["Charlie"])
	}
	seen := map[string]int{}
	for _, p := range []int{sectionDoing, sectionNext, sectionBlocked} {
		for _, id := range sectionIDs(m, p) {
			seen[id]++
		}
	}
	for _, title := range []string{"Alpha", "Bravo", "Charlie", "Delta", "Golf"} {
		if seen[ids[title]] != 1 {
			t.Fatalf("%s is in %d sections", title, seen[ids[title]])
		}
	}
	// Drawn blocked where it stands.
	view := ansi.Strip(m.overviewText())
	inProgress := view[:strings.Index(view, "NEXT UP")]
	if !strings.Contains(inProgress, bravo+"   blocked") {
		t.Fatalf("Bravo not drawn blocked in IN PROGRESS:\n%s", inProgress)
	}
	if strings.Count(view, bravo) != 1 {
		t.Fatalf("Bravo shown %d times in the overview:\n%s", strings.Count(view, bravo), view)
	}
}

// Check 2: the header counts are disjoint and add up to the tasks shown;
// doing names how many of its tasks are blocked.
func TestHeaderCountsAddUp(t *testing.T) {
	m, _ := placeBoard(t)
	header := strings.Split(ansi.Strip(m.View()), "\n")[2]
	want := "todo 2  ·  doing 2 (1 blocked)  ·  blocked 1  ·  done 1  ·  dropped 1"
	if strings.TrimSpace(header) != want {
		t.Fatalf("header %q, want %q", strings.TrimSpace(header), want)
	}
	sum := 0
	for _, part := range strings.Split(header, "·") {
		var name string
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(part), "%s %d", &name, &n); err != nil {
			t.Fatal(part, err)
		}
		sum += n
	}
	if sum != len(m.all) {
		t.Fatalf("counts sum to %d, there are %d tasks", sum, len(m.all))
	}
	// The same holds under a filter: counts cover the tasks shown.
	m = press(m, "/", "a", "enter")
	sum = 0
	for _, s := range boardStatuses {
		for _, v := range m.visible {
			if m.place(v) == s {
				sum++
			}
		}
	}
	if sum != len(m.visible) {
		t.Fatalf("filtered: places cover %d of %d", sum, len(m.visible))
	}
}

// Check 4: board columns and the status filter use the same one place.
func TestBoardAndStatusFilterOnePlace(t *testing.T) {
	m, ids := placeBoard(t)
	m.setView("board")
	total := 0
	for p, s := range boardStatuses {
		col := sectionIDs(m, p)
		total += len(col)
		switch s {
		case "doing":
			if !contains(col, ids["Bravo"]) || !contains(col, ids["Golf"]) {
				t.Fatalf("DOING column %v lacks Bravo or Golf", col)
			}
		case "blocked":
			if strings.Join(col, " ") != ids["Charlie"] {
				t.Fatalf("BLOCKED column %v, want only Charlie", col)
			}
		case "todo":
			if contains(col, ids["Charlie"]) {
				t.Fatal("blocked todo also in TODO")
			}
		}
	}
	if total != len(m.all) {
		t.Fatalf("board columns hold %d, there are %d tasks", total, len(m.all))
	}
	filtered := 0
	for _, s := range boardStatuses {
		n := m.valueCount("status", s)
		filtered += n
		m2 := commandLine(m, s)
		var got []string
		for _, v := range m2.visible {
			got = append(got, v.Title)
		}
		want := map[string]string{"todo": "Delta Alpha", "doing": "Bravo Golf", "blocked": "Charlie", "done": "Echo", "dropped": "Foxtrot"}[s]
		if strings.Join(got, " ") != want {
			t.Fatalf("status %s shows %v, want %s", s, got, want)
		}
	}
	if filtered != len(m.all) {
		t.Fatalf("status filter counts sum to %d, there are %d tasks", filtered, len(m.all))
	}
}

// closeBoard is a git repository on branch hug/tasks with tasks closed at
// known commit times, the way hug task done, set and edit close them.
func closeBoard(t *testing.T) (model, map[string]string, func(at time.Time, msg string)) {
	t.Helper()
	root := t.TempDir()
	run := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.invalid"), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run(nil, "init", "-q", "-b", "hug/tasks")
	commit := func(at time.Time, msg string) {
		t.Helper()
		date := fmt.Sprintf("%d +0000", at.Unix())
		run(nil, "add", "-A")
		run([]string{"GIT_AUTHOR_DATE=" + date, "GIT_COMMITTER_DATE=" + date}, "commit", "-q", "--allow-empty", "-m", msg)
	}
	s := &task.Store{Root: root}
	ids := map[string]string{}
	for _, title := range []string{"Old", "Kilo", "Lima", "Mike", "November", "Oscar"} {
		r, err := s.New(task.NewOptions{Title: title, Goal: "G", Acceptance: []string{"C"}})
		if err != nil {
			t.Fatal(err)
		}
		ids[title] = r.Task.ID
	}
	base := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	commit(base, "hug task new")
	done := func(title string) {
		t.Helper()
		if _, err := s.Check(ids[title], 1); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Done(ids[title]); err != nil {
			t.Fatal(err)
		}
	}
	// Old: done first, then a later note on the closed task (not a close).
	done("Old")
	commit(base.Add(1*time.Hour), "hug task done "+ids["Old"])
	// Kilo: closed by hug task done.
	done("Kilo")
	commit(base.Add(2*time.Hour), "hug task done "+ids["Kilo"])
	// Lima: closed by an edit of the status line (no done subject).
	d, _ := s.Get(ids["Lima"])
	path := filepath.Join(root, d.Task.Path)
	b, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), "status: todo", "status: done", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	commit(base.Add(3*time.Hour), "hug task edit "+ids["Lima"])
	// Mike: dropped.
	if _, err := s.Drop(ids["Mike"]); err != nil {
		t.Fatal(err)
	}
	commit(base.Add(4*time.Hour), "hug task drop "+ids["Mike"])
	// A note on Old, newest of all: touching a closed task is not closing it.
	d, _ = s.Get(ids["Old"])
	path = filepath.Join(root, d.Task.Path)
	b, _ = os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), "## Notes\n", "## Notes\nlate note\n", 1)), 0644); err != nil {
		t.Fatal(err)
	}
	commit(base.Add(5*time.Hour), "hug task note "+ids["Old"])
	m := newModel(s, Options{User: "tester"})
	m.width, m.height = 140, 40
	m.resize()
	return m, ids, commit
}

// Check 3: RECENTLY DONE lists every closed task, newest first, by the
// commit that closed them (h9zghj: all of them, not the last 3).
func TestRecentlyDoneNewestFirst(t *testing.T) {
	m, ids, commit := closeBoard(t)
	m.setView("overview")
	got := strings.Join(sectionIDs(m, sectionRecent), " ")
	want := strings.Join([]string{ids["Mike"], ids["Lima"], ids["Kilo"], ids["Old"]}, " ")
	if got != want {
		t.Fatalf("RECENTLY DONE %s, want Mike Lima Kilo Old %s (Old's late note is not a close)", got, want)
	}
	view := ansi.Strip(m.overviewText())
	at := strings.Index(view, "RECENTLY DONE · last closed  [4]")
	if at < 0 || at < strings.Index(view, "BLOCKED") || at > strings.Index(view, "LINT") {
		t.Fatalf("RECENTLY DONE not between BLOCKED and LINT:\n%s", view)
	}
	// A task closed since the last read comes first: only the new commits
	// are read.
	s := &task.Store{Root: m.store.Root}
	if _, err := s.Check(ids["November"], 1); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Done(ids["November"]); err != nil {
		t.Fatal(err)
	}
	commit(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), "hug task done "+ids["November"])
	m.refresh("")
	got = strings.Join(sectionIDs(m, sectionRecent)[:3], " ")
	want = strings.Join([]string{ids["November"], ids["Mike"], ids["Lima"]}, " ")
	if got != want {
		t.Fatalf("after a new close: %s, want November Mike Lima %s", got, want)
	}
}

// Check 3 at a small terminal: RECENTLY DONE gives way first.
func TestRecentlyDoneShrinksFirst(t *testing.T) {
	m := manyModel(t)
	m.width, m.height = 80, 24
	m.resize()
	m.setView("overview")
	rows := m.overviewRows()
	if rows[sectionRecent] != 1 {
		t.Fatalf("RECENTLY DONE got %d rows at 80x24, want 1 (it shrinks first): %v", rows[sectionRecent], rows)
	}
	m.height = 60
	m.resize()
	if rows = m.overviewRows(); rows[sectionRecent] != recentDone {
		t.Fatalf("RECENTLY DONE got %d rows at 80x60, want %d: %v", rows[sectionRecent], recentDone, rows)
	}
}

// mergeCloses reads subjects and status diffs, and ignores diffs of tasks
// that were already closed.
func TestMergeCloses(t *testing.T) {
	log := strings.Join([]string{
		"\x01300 hug task note aaaaaa",
		"diff --git a/h.aaaaaa.a.md b/h.aaaaaa.a.md",
		"+++ b/h.aaaaaa.a.md",
		"-status: done",
		"+status: \"done\"",
		"\x01200 hug task set bbbbbb",
		"diff --git a/g/h.bbbbbb.b.md b/g/h.bbbbbb.b.md",
		"+++ b/g/h.bbbbbb.b.md",
		"-status: \"doing\"",
		"+status: \"dropped\"",
		"\x01100 huginn done CCCCCC",
	}, "\n")
	got := map[string]int64{}
	mergeCloses(got, log)
	if len(got) != 2 || got["bbbbbb"] != 200 || got["cccccc"] != 100 {
		t.Fatalf("%v", got)
	}
}

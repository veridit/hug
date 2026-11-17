package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// setupBacklog500 is a repository with a 500-task backlog, imported.
func setupBacklog500(t *testing.T) *App {
	t.Helper()
	a := app(t)
	dir := filepath.Join(a.Cwd, ".backlog", "tasks")
	if e := os.MkdirAll(dir, 0o755); e != nil {
		t.Fatal(e)
	}
	for i := 1; i <= 500; i++ {
		body := fmt.Sprintf("---\nid: TASK-%d\ntitle: Task %d\nstatus: To Do\n---\n\n## Description\nGoal %d.\n\n## Acceptance Criteria\n- [ ] #1 It works\n", i, i, i)
		if e := os.WriteFile(filepath.Join(dir, fmt.Sprintf("task-%d - Task-%d.md", i, i)), []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	git(t, a.Cwd, "add", "-A")
	git(t, a.Cwd, "commit", "-q", "-m", "backlog")
	execute(t, a, "init")
	execute(t, a, "import", "backlog", ".backlog")
	return a
}

func setupStep(t *testing.T, a *App, name string) (SetupStep, string) {
	t.Helper()
	errb := &bytes.Buffer{}
	a.Err = errb
	for _, s := range execute(t, a, "setup")["setup"].([]SetupStep) {
		if s.Step == name {
			return s, errb.String()
		}
	}
	t.Fatalf("no %q step", name)
	return SetupStep{}, ""
}

// 3mavpt check 1: after an import, hug setup reads no backlog file. Every
// source file is made unreadable; the step still reports the import complete
// from the listing. On main setup re-ran the dry run, which fails reading
// them ("permission denied"), so the step stayed open.
func TestSetupReadsNoBacklogFileAfterImport(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("needs POSIX permissions that bind the user")
	}
	t.Parallel()
	a := setupBacklog500(t)
	files, _ := filepath.Glob(filepath.Join(a.Cwd, ".backlog", "tasks", "*.md"))
	if len(files) != 500 {
		t.Fatalf("%d files", len(files))
	}
	for _, f := range files {
		if e := os.Chmod(f, 0); e != nil {
			t.Fatal(e)
		}
	}
	t.Cleanup(func() {
		for _, f := range files {
			os.Chmod(f, 0o644)
		}
	})
	s, stderr := setupStep(t, a, "import backlog-md")
	if !s.Done || !strings.Contains(s.Detail, "0 to add, 0 to update, 500 in sync") || !strings.Contains(s.Detail, "Remembered from ") || strings.Contains(stderr, "comparing") {
		t.Fatalf("setup read the backlog: %+v\nstderr: %s", s, stderr)
	}
}

// A changed backlog is compared again, and says so: an edited file, a new
// file, and a task gone from the board each send setup back to the dry run.
func TestSetupComparesAgainWhenBacklogOrBoardChanges(t *testing.T) {
	t.Parallel()
	a := setupBacklog500(t)
	dir := filepath.Join(a.Cwd, ".backlog", "tasks")
	if s, stderr := setupStep(t, a, "import backlog-md"); !s.Done || strings.Contains(stderr, "comparing") {
		t.Fatalf("after import: %+v %s", s, stderr)
	}
	// Edited upstream: same size, later mtime is enough to look again.
	p := filepath.Join(dir, "task-7 - Task-7.md")
	b, _ := os.ReadFile(p)
	if e := os.WriteFile(p, bytes.Replace(b, []byte("Goal 7."), []byte("Goal 8."), 1), 0o644); e != nil {
		t.Fatal(e)
	}
	later := time.Now().Add(2 * time.Second)
	os.Chtimes(p, later, later)
	s, stderr := setupStep(t, a, "import backlog-md")
	if s.Done || !strings.Contains(s.Detail, "0 to add, 1 to update, 499 in sync") || !strings.Contains(stderr, "hug setup: comparing .backlog/ with the board") {
		t.Fatalf("edited source: %+v\nstderr: %s", s, stderr)
	}
	// The dry run is remembered too: the next setup reads nothing.
	if s, stderr = setupStep(t, a, "import backlog-md"); s.Done || !strings.Contains(s.Detail, "1 to update") || strings.Contains(stderr, "comparing") {
		t.Fatalf("second setup: %+v %s", s, stderr)
	}
	execute(t, a, "import", "backlog", ".backlog")
	// A new backlog file.
	if e := os.WriteFile(filepath.Join(dir, "task-501 - New.md"), []byte("---\nid: TASK-501\ntitle: New\nstatus: To Do\n---\n\nNew.\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	if s, stderr = setupStep(t, a, "import backlog-md"); s.Done || !strings.Contains(s.Detail, "1 to add, 0 to update, 500 in sync") || !strings.Contains(stderr, "comparing") {
		t.Fatalf("new source: %+v %s", s, stderr)
	}
	execute(t, a, "import", "backlog", ".backlog")
	// A task removed from the board (its file deleted and committed).
	m, _ := filepath.Glob(filepath.Join(a.Cwd, TaskDir, "*task-501*.md"))
	if len(m) == 0 {
		m, _ = filepath.Glob(filepath.Join(a.Cwd, TaskDir, "*new*.md"))
	}
	if len(m) != 1 {
		t.Fatalf("board file for TASK-501: %v", m)
	}
	board := filepath.Join(a.Cwd, TaskDir)
	git(t, board, "rm", "-q", "--", filepath.Base(m[0]))
	git(t, board, "-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "drop TASK-501")
	if s, stderr = setupStep(t, a, "import backlog-md"); s.Done || !strings.Contains(stderr, "comparing") || !strings.Contains(s.Detail, "1 to add") {
		t.Fatalf("task gone from board: %+v %s", s, stderr)
	}
}

// cancelOnWrite cancels a context on the first write: setup is interrupted
// the moment it says it starts the dry run.
type cancelOnWrite struct{ cancel context.CancelFunc }

func (c cancelOnWrite) Write(p []byte) (int, error) { c.cancel(); return len(p), nil }

// 3mavpt check 3: an interrupted setup says so, never a checklist whose
// steps failed because their git calls were killed (Seid, ^C: "Model:
// trailer gate: the commit-msg gate needs a Git checkout as its working
// directory" in seid-lang's own checkout).
func TestSetupInterruptedSaysSo(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e := a.Execute(ctx, []string{"setup"})
	if e == nil || !strings.Contains(e.Error(), "hug task setup was interrupted") {
		t.Fatalf("cancelled before start: %v", e)
	}
	// Cancelled in the middle, during the backlog dry run.
	b := setupBacklog500(t)
	f := filepath.Join(b.Cwd, ".backlog", "tasks", "task-1 - Task-1.md")
	later := time.Now().Add(2 * time.Second)
	os.Chtimes(f, later, later)
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	b.Err = cancelOnWrite{cancel}
	r, e := b.Execute(ctx, []string{"setup"})
	if e == nil || !strings.Contains(e.Error(), "hug task setup was interrupted") {
		t.Fatalf("cancelled during the dry run: %v %v", e, r)
	}
	if _, _, fix := ErrorDetails(e); fix != "hug task setup" {
		t.Fatalf("fix: %q", fix)
	}
}

// The task-files step is remembered too, keyed on the folder's files and
// the board's commit: a new evidence file, or a board commit, is counted.
func TestSetupFilesStepRemembered(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Has evidence", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	ev := filepath.Join(a.Cwd, "evidence", id)
	if e := os.MkdirAll(ev, 0o755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(ev, "run.log"), []byte("ok\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	step := func() string { s, _ := setupStep(t, a, "import task files"); return s.Detail }
	if d := step(); !strings.Contains(d, "1 files to copy, 0 already in place, 0 citations") {
		t.Fatal(d)
	}
	// Remembered: the same answer with the board file unreadable is not a
	// proof here (ImportFiles reads the board), so check the cache file.
	m, _ := filepath.Glob(filepath.Join(a.Cwd, ".git", "huginn", "setup-files-*.json"))
	if len(m) != 1 {
		t.Fatalf("remembered files check: %v", m)
	}
	if e := os.WriteFile(filepath.Join(ev, "second.log"), []byte("ok\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	if d := step(); !strings.Contains(d, "2 files to copy") {
		t.Fatalf("new evidence file: %s", d)
	}
	execute(t, a, "note", id, "see evidence/"+id+"/run.log")
	if d := step(); !strings.Contains(d, "1 citations to rewrite") {
		t.Fatalf("board commit: %s", d)
	}
}

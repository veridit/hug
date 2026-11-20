package task

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	s := &Store{Root: t.TempDir(), DepsMode: "warn"}
	r, err := s.New(NewOptions{Title: "Bench", Goal: "measure", Acceptance: []string{"numbers"}})
	if err != nil {
		t.Fatal(err)
	}
	return s, r.Task.ID
}

func TestTaskFiles(t *testing.T) {
	s, id := newStore(t)
	// attach, then read back the identical bytes
	if _, ar, err := s.Attach(id, "bench/runs.txt", []byte("12ms\n"), "baseline", false); err != nil || !ar.Changed {
		t.Fatalf("attach: %+v %v", ar, err)
	}
	if b, err := s.ReadFile(id, "bench/runs.txt"); err != nil || string(b) != "12ms\n" {
		t.Fatalf("read back %q %v", b, err)
	}
	// identical re-attach changes nothing; other content needs replace
	if _, ar, _ := s.Attach(id, "bench/runs.txt", []byte("12ms\n"), "", false); ar.Changed {
		t.Error("identical re-attach changed something")
	}
	if _, _, err := s.Attach(id, "bench/runs.txt", []byte("13ms\n"), "", false); err == nil {
		t.Error("different content without replace was accepted")
	}
	if _, ar, err := s.Attach(id, "bench/runs.txt", []byte("13ms\n"), "", true); err != nil || ar.Old == "" {
		t.Fatalf("replace: %+v %v", ar, err)
	}
	// binary and oversized are refused and nothing is written
	for name, b := range map[string][]byte{"x.bin": {'a', 0, 'b'}, "big.txt": []byte(strings.Repeat("x", MaxFileBytes+1))} {
		if _, _, err := s.Attach(id, name, b, "", false); err == nil {
			t.Errorf("%s accepted", name)
		}
		if _, err := s.ReadFile(id, name); err == nil {
			t.Errorf("%s written despite refusal", name)
		}
	}
	// unsafe names
	for _, n := range []string{"../escape.txt", "/abs.txt", ".hidden", "a/../b.txt"} {
		if _, _, err := s.Attach(id, n, []byte("x"), "", false); err == nil {
			t.Errorf("unsafe name %q accepted", n)
		}
	}
	// the files folder holds files, never tasks: a .md inside it is not a task
	if _, _, err := s.Attach(id, "notes.md", []byte("# not a task\n"), "", false); err != nil {
		t.Fatal(err)
	}
	ts, err := s.Tasks()
	if err != nil || len(ts) != 1 {
		t.Fatalf("tasks = %d %v; files folder was read as tasks", len(ts), err)
	}
	// moving the task to a group keeps it and its folder where they are
	// (w40cmk check 3: the group is a field, the path follows from the id)
	if _, err := s.MoveGroup(id, "perf"); err != nil {
		t.Fatal(err)
	}
	if b, err := s.ReadFile(id, "bench/runs.txt"); err != nil || string(b) != "13ms\n" {
		t.Fatalf("after move: %q %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, filepath.FromSlash(FolderFor(id)), "notes.md")); err != nil {
		t.Errorf("folder left C/ID/ on a group move: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Root, "perf")); !os.IsNotExist(err) {
		t.Errorf("a group move made a perf/ folder: %v", err)
	}
	// lint: a missing reference, and a hand-placed binary file
	s.Note(id, "see file:missing.tsv and task:"+id+"/bench/runs.txt; prose like file:line is no reference")
	os.WriteFile(filepath.Join(s.Root, filepath.FromSlash(FolderFor(id)), "hand.bin"), []byte{0, 1}, 0o644)
	fs, err := s.LintFiles([]string{".jcode/scratch/"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, f.Problem)
	}
	all := strings.Join(got, "\n")
	if !strings.Contains(all, "file:missing.tsv: file not found") || !strings.Contains(all, "file-not-text") || strings.Contains(all, "file:line") || strings.Contains(all, "bench/runs.txt") {
		t.Errorf("lint findings:\n%s", all)
	}
	// detach removes the file and notes it
	if _, err := s.Detach(id, "bench/runs.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ReadFile(id, "bench/runs.txt"); err == nil {
		t.Error("detached file still readable")
	}
}

// TestForbidCiteNamesPathAndLine (cvbpdy check 1): each citation of a
// forbidden path is its own finding with the cited path and its line; a
// citation followed by " (lost)" or " (not kept: ...)" (also after a
// closing backtick) is none, and an absolute path that only contains the
// prefix names another folder (as Seid's tools/evidence.py reads them).
func TestForbidCiteNamesPathAndLine(t *testing.T) {
	s, id := newStore(t)
	s.Note(id, "evidence in .jcode/scratch/T1/run-{a,b}.log and `.jcode/scratch/T2/old.log` (lost); .jcode/scratch/T3/gone.txt (not kept: run output), home /Users/x/.jcode/scratch/T4/y.md")
	fs, err := s.LintFiles([]string{".jcode/scratch/"})
	if err != nil {
		t.Fatal(err)
	}
	var cites []Finding
	for _, f := range fs {
		if FindingCode(f.Problem) == "forbidden-citation" {
			cites = append(cites, f)
		}
	}
	if len(cites) != 1 {
		t.Fatalf("want 1 forbidden-citation (T1), got %d: %+v", len(cites), cites)
	}
	f := cites[0]
	doc, _ := os.ReadFile(filepath.Join(s.Root, filepath.FromSlash(f.Path)))
	lines := strings.Split(string(doc), "\n")
	if !strings.Contains(f.Problem, "cites .jcode/scratch/T1/run-{a,b}.log (") || f.Line < 1 || f.Line > len(lines) || !strings.Contains(lines[f.Line-1], "T1/run-") {
		t.Errorf("finding %q at line %d; want the path and the line that holds it", f.Problem, f.Line)
	}
	if !strings.Contains(f.Problem, " (lost)") {
		t.Errorf("finding %q does not say how to mark it lost", f.Problem)
	}
}

// TestTaskPathCitationMustExist (cvbpdy check 2, narrowed by 7j706s): a
// file: link that does not exist is missing-file; an existing one and one
// marked " (lost)" are not. A .hug/tasks/<id>/NAME path is quoted text, not
// a link: no finding, whether or not the file is there.
func TestTaskPathCitationMustExist(t *testing.T) {
	s, id := newStore(t)
	if _, _, err := s.Attach(id, "report.md", []byte("ok\n"), "", false); err != nil {
		t.Fatal(err)
	}
	s.Note(id, "see file:report.md and (file:missing.tsv), also file:old.tsv (lost); quoted: `.hug/tasks/"+id+"/quoted.tsv`")
	fs, err := s.LintFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	var missing []string
	for _, f := range fs {
		if FindingCode(f.Problem) == "missing-file" {
			missing = append(missing, f.Problem+" | fix: "+f.FixLine())
		}
	}
	if len(missing) != 1 || !strings.HasPrefix(missing[0], "file:missing.tsv: file not found") || !strings.Contains(missing[0], "hug task attach "+id+" FILE --name missing.tsv") {
		t.Errorf("missing-file findings: %q; want exactly missing.tsv with an attach fix", missing)
	}
}

// TestDetachLeavesNoMissingFile (bqfrda check 1; piglet 2026-11-18 on
// gpyqq9): after attach and detach the attach note's file:NAME and the
// detach note's "detached file:NAME" stayed citations, and lint reported
// the same missing file twice. The detach says the file is gone on
// purpose: no missing-file for it. Attached again, it is checked again.
func TestDetachLeavesNoMissingFile(t *testing.T) {
	s, id := newStore(t)
	missing := func() []string {
		t.Helper()
		fs, err := s.LintFiles(nil)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, f := range fs {
			if FindingCode(f.Problem) == "missing-file" {
				out = append(out, f.Problem)
			}
		}
		return out
	}
	if _, _, err := s.Attach(id, "ev.txt", []byte("x\n"), "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Detach(id, "ev.txt"); err != nil {
		t.Fatal(err)
	}
	if got := missing(); len(got) != 0 {
		t.Fatalf("after attach and detach: want no missing-file, got %q", got)
	}
	if _, _, err := s.Attach(id, "ev.txt", []byte("y\n"), "", false); err != nil {
		t.Fatal(err)
	}
	if got := missing(); len(got) != 0 {
		t.Fatalf("attached again: want no missing-file, got %q", got)
	}
	// Attached again, then removed by hand: that citation is checked again.
	p, err := s.diskPath(FolderOf(Task{ID: id, Path: taskPath(t, s, id)}) + "/ev.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(p); err != nil {
		t.Fatal(err)
	}
	if got := missing(); len(got) != 1 || !strings.HasPrefix(got[0], "file:ev.txt: file not found") {
		t.Fatalf("re-attached, then removed by hand: want one missing-file for ev.txt, got %q", got)
	}
}

// TestMissingFileReportedOnce (bqfrda check 2): a cited file that is not
// there is one finding, however often the task cites it.
func TestMissingFileReportedOnce(t *testing.T) {
	s, id := newStore(t)
	s.Note(id, "see file:gone.tsv")
	s.Note(id, "and again file:gone.tsv, also .hug/tasks/"+id+"/gone.tsv")
	fs, err := s.LintFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		if FindingCode(f.Problem) == "missing-file" {
			got = append(got, f.Problem+" | fix: "+f.FixLine())
		}
	}
	if len(got) != 1 || !strings.Contains(got[0], "gone.tsv: file not found") || !strings.HasSuffix(got[0], "fix: hug task attach "+id+" FILE --name gone.tsv") {
		t.Fatalf("one missing file cited 3 times: want one finding with the attach fix, got %q", got)
	}
}

func taskPath(t *testing.T, s *Store, id string) string {
	t.Helper()
	ts, err := s.Tasks()
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range ts {
		if x.ID == id {
			return x.Path
		}
	}
	t.Fatalf("no task %s", id)
	return ""
}

// TestQuotedPathOfAnotherBoardIsNoLink (1phmre check 1; w40cmk's note
// quotes Statbus's '.hug/tasks/37bryg/README.md' from a design talk): a
// .hug/tasks/ID/NAME path is quoted text, not a link: no missing-task-ref,
// no missing-file, for another board's id or ours (7j706s). A file: link of
// ours is still checked.
func TestQuotedPathOfAnotherBoardIsNoLink(t *testing.T) {
	s, id := newStore(t)
	s.Note(id, "Proposal (crab): one folder per task: .hug/tasks/37bryg/README.md is the task, 'its files sit beside it (.hug/tasks/37bryg/STATBUS-369.md)'")
	s.Note(id, "and ours: .hug/tasks/"+id+"/quoted.tsv, file:gone.tsv")
	fs, err := s.LintFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, FindingCode(f.Problem)+": "+f.Problem)
	}
	if len(got) != 1 || !strings.HasPrefix(got[0], "missing-file: file:gone.tsv: file not found") {
		t.Fatalf("quoted Statbus paths and one missing file of ours: want only the missing-file, got %q", got)
	}
}

// TestMissingTaskLinkFixIsNonInteractive (1phmre check 2): a task:REF/NAME
// link to an id no task has is still missing-task-ref, and its fix is a
// command an agent runs without an editor: the note that marks the link
// (lost). After that note, lint has no finding for it.
func TestMissingTaskLinkFixIsNonInteractive(t *testing.T) {
	s, id := newStore(t)
	s.Note(id, "evidence in task:zz9zz9/run.log")
	fs, err := s.LintFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || FindingCode(fs[0].Problem) != "missing-task-ref" || fs[0].Problem != "task:zz9zz9/run.log: no task zz9zz9 (fix the id, or mark the link (lost))" {
		t.Fatalf("a task: link to no task: %+v", fs)
	}
	fix := fs[0].FixLine()
	if want := `hug task note ` + id + ` "task:zz9zz9/run.log (lost)"`; fix != want {
		t.Fatalf("fix %q, want the non-interactive %q", fix, want)
	}
	// The fix, as hug task note runs it.
	if _, err := s.Note(id, "task:zz9zz9/run.log (lost)"); err != nil {
		t.Fatal(err)
	}
	if fs, err = s.LintFiles(nil); err != nil || len(fs) != 0 {
		t.Fatalf("after the fix: %+v %v", fs, err)
	}
}

// ydwamb check 1 (911j70's note: "open -a Preview
// .hug/tasks/9/911j70/compare-*.png"): a citation whose last path part has
// a shell glob (* ? [..]) is the pattern it is, matched against the task's
// attached files. Two compare-*.png attached: no finding. Lint cut it at
// the * before and reported ".../compare-: file not found".
func TestGlobCitationMatchingFilesIsFine(t *testing.T) {
	s, id := newStore(t)
	for _, name := range []string{"compare-80.png", "compare-140.png"} {
		if _, _, err := s.Attach(id, name, png1x1, "", false); err != nil {
			t.Fatal(err)
		}
	}
	tk, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	folder := FolderOf(tk.Task)
	s.Note(id, "Jørgen: open -a Preview .hug/tasks/"+folder+"/compare-*.png")
	s.Note(id, "also `.hug/tasks/"+folder+"/compare-1?0.png`, .hug/tasks/"+folder+"/compare-[18]*.png, file:compare-*.png and task:"+id+"/compare-*.png")
	fs, err := s.LintFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatalf("glob citations that match attached files: want no finding, got %+v", fs)
	}
}

// ydwamb check 2: a glob citation that matches no attached file is one
// missing-file finding naming the whole pattern, and its fix attaches a
// file under no cut-off name; a plain missing file is still found.
func TestGlobCitationMatchingNothingNamesThePattern(t *testing.T) {
	s, id := newStore(t)
	if _, _, err := s.Attach(id, "compare-80.png", png1x1, "", false); err != nil {
		t.Fatal(err)
	}
	tk, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	folder := FolderOf(tk.Task)
	s.Note(id, "see file:shot-*.png and task:"+id+"/gone.tsv; quoted, no link: .hug/tasks/"+folder+"/other-*.png")
	fs, err := s.LintFiles(nil)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, f := range fs {
		got = append(got, FindingCode(f.Problem)+": "+f.Problem+" | fix: "+f.FixLine())
	}
	if len(got) != 2 {
		t.Fatalf("want 2 missing-file findings (the pattern, gone.tsv), got %q", got)
	}
	glob := ""
	for _, g := range got {
		if strings.Contains(g, "shot-") {
			glob = g
		}
	}
	if !strings.HasPrefix(glob, "missing-file: file:shot-*.png: no attached file matches") {
		t.Fatalf("the pattern finding: %q", glob)
	}
	if strings.Contains(glob, "--name shot-") || strings.Contains(glob, "--name shot-*") {
		t.Fatalf("the fix attaches under a cut-off or glob name: %q", glob)
	}
	if !strings.Contains(glob, "fix: hug task attach "+id+" FILE") {
		t.Fatalf("the fix is not an attach: %q", glob)
	}
	if !slicesContains(got, "missing-file: task:"+id+"/gone.tsv: file not found") {
		t.Fatalf("a plain missing file is no longer found: %q", got)
	}
}

// png1x1 is a 1x1 PNG: an image is a task file when its name says so.
var png1x1 = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82")

// slicesContains says whether a string in list starts with prefix.
func slicesContains(list []string, prefix string) bool {
	for _, s := range list {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// ydwamb: glob characters are read only as a pattern; prose around a plain
// citation stays out of the name, as before.
func TestCitedNameKeepsProseOut(t *testing.T) {
	for in, want := range map[string]string{
		"a.md":            "a.md",
		"a.md?":           "a.md",
		"a.md**":          "a.md",
		"a.md]":           "a.md",
		"a.md?raw=1":      "a.md",
		"compare-*.png":   "compare-*.png",
		"compare-1?0.png": "compare-1?0.png",
		"c-[18]*.png":     "c-[18]*.png",
		"logs/run-*.log":  "logs/run-*.log",
		"a[b":             "",
		"*/x.md":          "",
	} {
		got, ok := citedName(in)
		if want == "" {
			if ok {
				t.Errorf("citedName(%q) = %q, want no citation", in, got)
			}
			continue
		}
		if !ok || got != want {
			t.Errorf("citedName(%q) = %q %v, want %q", in, got, ok, want)
		}
	}
}

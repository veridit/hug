package gitbranch

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// replayPair is a remote board moved to layout 2 (every file under b/,
// hug.toml "2") and a clone with n unpushed commits on layout 1, each
// note-appending to its own file. The translator maps a/NAME to b/NAME.
func replayPair(t *testing.T, n int) (*Repository, Translator) {
	t.Helper()
	a, b := pair(t)
	for i := 0; i < 3; i++ {
		writeFile(t, a, fmt.Sprintf("a/t%d.md", i), "task "+strconv.Itoa(i)+"\n")
	}
	writeFile(t, a, "hug.toml", "1\n")
	must(t, a.Commit(nil, "board on layout 1"))
	must(t, a.Sync())
	must(t, b.Sync())
	// The remote moves the board.
	for i := 0; i < 3; i++ {
		p := fmt.Sprintf("t%d.md", i)
		must(t, os.MkdirAll(filepath.Join(a.Dir, "b"), 0o755))
		must(t, os.Rename(filepath.Join(a.Dir, "a", p), filepath.Join(a.Dir, "b", p)))
	}
	writeFile(t, a, "hug.toml", "2\n")
	must(t, a.Commit(nil, "move to layout 2"))
	must(t, a.Sync())
	// The clone, still on layout 1, makes n commits.
	for i := 0; i < n; i++ {
		p := fmt.Sprintf("a/t%d.md", i%3)
		raw, e := os.ReadFile(filepath.Join(b.Dir, p))
		must(t, e)
		writeFile(t, b, p, string(raw)+"note "+strconv.Itoa(i)+"\n")
		must(t, b.Commit(nil, "note "+strconv.Itoa(i)))
	}
	tr := Translator{
		Names: func(paths []string) map[string]string {
			out := map[string]string{}
			for _, p := range paths {
				out[p] = strings.Replace(p, "a/", "b/", 1)
			}
			return out
		},
		Content: func(_, _ string, b []byte) []byte { return b },
		Merge: func(_ string, base, ours, theirs []byte, _ Replayed) ([]byte, []string, error) {
			if bytes.Contains(ours, []byte("unreadable")) {
				return nil, nil, errors.New("hug cannot read this file")
			}
			if theirs == nil || bytes.Equal(theirs, base) {
				return ours, nil, nil
			}
			// Both sides appended: theirs, then our new lines.
			return append(append([]byte{}, theirs...), bytes.TrimPrefix(ours, base)...), nil, nil
		},
		Layout: func(text string) (int, error) {
			if strings.TrimSpace(text) == "" {
				return 1, nil
			}
			return strconv.Atoi(strings.TrimSpace(text))
		},
	}
	b.Layout = tr.Layout
	must(t, b.fetchOnly())
	return b, tr
}

// fetchOnly brings the remote ref up to date without merging.
func (r *Repository) fetchOnly() error {
	_, e := r.fetch()
	return e
}

// gitStarts counts the git processes trace2 saw this process start in
// dir, by name: a git's own children (maintenance, upload-pack) have a
// "/" in their sid and are not counted.
func gitStarts(t *testing.T, dir string) (total int, by map[string]int) {
	t.Helper()
	by = map[string]int{}
	ents, e := os.ReadDir(dir)
	if e != nil {
		return 0, by
	}
	for _, f := range ents {
		raw, e := os.ReadFile(filepath.Join(dir, f.Name()))
		must(t, e)
		for _, l := range strings.Split(string(raw), "\n") {
			if !strings.Contains(l, `"event":"start"`) {
				continue
			}
			if sid, _, _ := strings.Cut(l[strings.Index(l, `"sid":"`)+7:], `"`); strings.Contains(sid, "/") {
				continue
			}
			total++
			i := strings.Index(l, `"argv":[`)
			args := strings.Split(l[i+8:], ",")
			name := "?"
			for j := 1; j < len(args); j++ {
				a := strings.Trim(args[j], `"]}`)
				if a == "-C" || a == "-c" {
					j++
					continue
				}
				name = a
				break
			}
			by[name]++
		}
	}
	return total, by
}

// 0n4rnx: a replay of N commits starts at most 5N+20 git processes; one
// object reader per call, not a git per tree, file and parent (it was
// about 13 per commit touching one file).
func TestCatchUpProcessesPerCommit(t *testing.T) {
	const n = 12
	r, tr := replayPair(t, n)
	trace := t.TempDir()
	t.Setenv("GIT_TRACE2_EVENT", trace)
	list, e := r.CatchUp(tr)
	t.Setenv("GIT_TRACE2_EVENT", "")
	must(t, e)
	if len(list) != n {
		t.Fatalf("replayed %d commits, want %d", len(list), n)
	}
	for i := 0; i < 3; i++ {
		got, e := os.ReadFile(filepath.Join(r.Dir, fmt.Sprintf("b/t%d.md", i)))
		must(t, e)
		if want := n / 3; strings.Count(string(got), "note ") != want {
			t.Fatalf("b/t%d.md has %d notes, want %d:\n%s", i, strings.Count(string(got), "note "), want, got)
		}
	}
	total, by := gitStarts(t, trace)
	t.Logf("catch-up of %d commits: %d git processes %v", n, total, by)
	if total > 5*n+20 {
		t.Fatalf("catch-up of %d commits started %d git processes, want at most %d: %v", n, total, 5*n+20, by)
	}
}

// 0n4rnx: a replay that stops leaves no git object reader running: every
// cat-file --batch the trace saw start also exited.
func TestCatchUpStopLeavesNoReader(t *testing.T) {
	r, tr := replayPair(t, 4)
	p := filepath.Join(r.Dir, "a", "t1.md")
	raw, e := os.ReadFile(p)
	must(t, e)
	writeFile(t, r, "a/t1.md", string(raw)+"unreadable\n")
	must(t, r.Commit(nil, "a file the merge cannot read"))
	head, e := r.Head()
	must(t, e)
	trace := t.TempDir()
	t.Setenv("GIT_TRACE2_EVENT", trace)
	_, e = r.CatchUp(tr)
	t.Setenv("GIT_TRACE2_EVENT", "")
	var stopped *CatchUpError
	if !errors.As(e, &stopped) || stopped.Path != "b/t1.md" {
		t.Fatalf("catch-up should stop at b/t1.md: %v", e)
	}
	if now, _ := r.Head(); now != head {
		t.Fatalf("HEAD moved on a stopped replay: %s -> %s", head, now)
	}
	readers, open := 0, 0
	ents, _ := os.ReadDir(trace)
	for _, f := range ents {
		b, e := os.ReadFile(filepath.Join(trace, f.Name()))
		must(t, e)
		if bytes.Contains(b, []byte(`"cat-file","--batch"`)) {
			readers++
			if !bytes.Contains(b, []byte(`"event":"exit"`)) {
				open++
			}
		}
	}
	if readers == 0 {
		t.Fatal("no cat-file --batch reader started: the test does not show the reader's end")
	}
	if open != 0 {
		t.Fatalf("%d of %d object readers did not exit after a stopped replay", open, readers)
	}
}

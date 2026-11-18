// Package repocheck holds tests that judge the repository's files
// themselves, not any one package's code.
package repocheck

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// withMarkers lists tracked files that hold conflict markers on purpose,
// by path, never by pattern (55a7xx). None do today.
var withMarkers = map[string]bool{}

// marker reports whether line is a merge conflict marker: <<<<<<<,
// ||||||| (diff3 base) and >>>>>>> start one; ======= is one only in a
// file that has another, because a 7-character Markdown heading is
// underlined with exactly seven.
func marker(line string) (kind string, strong bool) {
	for _, m := range []string{"<<<<<<<", "|||||||", ">>>>>>>"} {
		if line == m || strings.HasPrefix(line, m+" ") {
			return m, true
		}
	}
	if line == "=======" {
		return line, false
	}
	return "", false
}

// 55a7xx: no tracked text file holds a leftover merge conflict marker. On
// 2026-11-18 two rebases left '||||||| parent of ...' lines in CHANGELOG.md
// on main (27643306, 1151adad). One git ls-files call, then a read of each
// file; binary files are skipped.
func TestNoConflictMarkers(t *testing.T) {
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("not in a git checkout: %v", err)
	}
	root := strings.TrimSpace(string(top))
	list := exec.Command("git", "ls-files", "-z")
	list.Dir = root
	out, err := list.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	files := 0
	for _, rel := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if rel == "" || withMarkers[rel] {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue // deleted in the worktree, or a submodule
		}
		if bytes.IndexByte(b[:min(len(b), 8000)], 0) >= 0 {
			continue // binary
		}
		files++
		var weak []int
		strong := false
		for i, line := range strings.Split(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n") {
			kind, s := marker(line)
			switch {
			case s:
				strong = true
				t.Errorf("%s:%d: leftover conflict marker %q\nfix: resolve the conflict in %s and delete line %d", rel, i+1, kind, rel, i+1)
			case kind != "":
				weak = append(weak, i+1)
			}
		}
		if strong {
			for _, n := range weak {
				t.Errorf("%s:%d: leftover conflict marker \"=======\"\nfix: resolve the conflict in %s and delete line %d", rel, n, rel, n)
			}
		}
	}
	if files == 0 {
		t.Fatal("read no files: git ls-files listed nothing readable")
	}
}

func TestMarkerLines(t *testing.T) {
	for line, want := range map[string]bool{
		"<<<<<<< HEAD": true, "||||||| parent of c9ad1b2e (x)": true, ">>>>>>> c9ad1b2e (x)": true,
		"|||||||": true, "=======": false, "==================================": false,
		"<<<<<<<<": false, " <<<<<<< HEAD": false, "a ||||||| b": false,
	} {
		if _, got := marker(line); got != want {
			t.Errorf("marker(%q) strong = %v, want %v", line, got, want)
		}
	}
	if kind, _ := marker("======="); kind == "" {
		t.Error("======= is not seen at all")
	}
}

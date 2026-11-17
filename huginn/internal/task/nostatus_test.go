package task

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// 6ephqk step 4 (crab 11:10): after the rename, no source reads or writes
// a task's state under the word "status". The only reads left are the
// upgrade's migration and things that are not a task's state: git status,
// hug apps' install status, hug mem model status, and the foreign files the
// importer reads (Backlog.md's own status field). Each is listed with why.
func TestNoStatusLeft(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(here), "..", "..", "..")
	allowed := []struct{ file, pattern, why string }{
		{"huginn/internal/task/stateupgrade.go", ``, "the migration: the one reader of the old key"},
		{"huginn/gitbranch/", `"status", "--porcelain"|fix\("status"\)`, "git status"},
		{"muninn/internal/project/branch.go", `"status", "--porcelain"`, "git status"},
		{"cmd/hug/apps.go", ``, "hug apps: an app's install status (ok, missing, stale)"},
		{"muninn/run/model.go", `Use: "status"`, "hug mem model status"},
		{"huginn/internal/importer/importer.go", `s\.fields\["status"\]`, "Backlog.md's own status field, read on import"},
		{"huginn/internal/importer/github.go", ``, "GitHub issue state, read on import"},
	}
	word := regexp.MustCompile(`"status["\s:]|\bstatus:\s|--status\b|json:"status`)
	var left []string
	e := filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if n := d.Name(); n == ".git" || n == ".hug" || n == "testdata" || n == "node_modules" || n == "tmp" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for n := 1; sc.Scan(); n++ {
			line := sc.Text()
			if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "//") || !word.MatchString(line) {
				continue
			}
			ok := false
			for _, a := range allowed {
				if strings.HasPrefix(rel, a.file) && (a.pattern == "" || regexp.MustCompile(a.pattern).MatchString(line)) {
					ok = true
					break
				}
			}
			if !ok {
				left = append(left, rel+":"+itoa(n)+": "+strings.TrimSpace(line))
			}
		}
		return sc.Err()
	})
	if e != nil {
		t.Fatal(e)
	}
	if len(left) > 0 {
		t.Fatalf("%d lines still say status for a task's state (6ephqk step 4: the word is state):\n%s", len(left), strings.Join(left, "\n"))
	}
}


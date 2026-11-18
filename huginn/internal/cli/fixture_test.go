package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Fixture templates (0n4rnx): a fixture several tests share is built once
// per test process, under fixtureRoot, and each test gets its own copy. On
// the Windows runner a git process costs ~34 ms; clones() alone started 77
// of them and the catch-up pair (clones, a layout-2 board, push and pull)
// 117, in each of 30 tests. A copy starts none: the template's absolute
// paths (remote url, worktree links, FETCH_HEAD, logs) are rewritten in
// Go, and a copy that still names the template fails its test.

// fixtureRoot holds every template; TestMain removes it after the run.
var (
	fixtureRoot string
	fixtureMu   sync.Mutex
	fixtures    = map[string]*template{}
)

type template struct {
	sync.Mutex
	dir    string
	failed bool
}

// fixture copies template key into a fresh t.TempDir and returns the copy.
// The first caller builds the template with build, into an empty folder
// that outlives the test; build must not use t.TempDir or t.Cleanup for
// what it leaves there. A build that failed fails every later caller too.
func fixture(t *testing.T, key string, build func(t *testing.T, dir string)) string {
	t.Helper()
	fixtureMu.Lock()
	if fixtureRoot == "" {
		root, e := os.MkdirTemp("", "hug-fixture-")
		if e != nil {
			fixtureMu.Unlock()
			t.Fatal(e)
		}
		fixtureRoot = root
	}
	tp := fixtures[key]
	if tp == nil {
		tp = &template{}
		fixtures[key] = tp
	}
	root := fixtureRoot
	fixtureMu.Unlock()

	tp.Lock()
	if tp.failed {
		tp.Unlock()
		t.Fatalf("fixture %s failed to build in an earlier test", key)
	}
	if tp.dir == "" {
		dir := filepath.Join(root, key)
		tp.failed = true // until build returns
		func() {
			defer tp.Unlock()
			build(t, dir)
			tp.failed, tp.dir = false, dir
		}()
	} else {
		tp.Unlock()
	}
	dst := t.TempDir()
	copyFixture(t, tp.dir, dst, filepath.Base(root))
	return dst
}

// pathPairs maps each way git and hug write src into a file to the same
// way for dst: as given, with forward slashes (git on Windows), with
// backslashes doubled (a Windows path in git config, replaced first), and
// with symlinks resolved (macOS /var is /private/var; Windows short names).
func pathPairs(src, dst string) (from, to []string) {
	add := func(s, d string) {
		from = append(from, strings.ReplaceAll(s, `\`, `\\`), s, filepath.ToSlash(s))
		to = append(to, strings.ReplaceAll(d, `\`, `\\`), d, filepath.ToSlash(d))
	}
	add(src, dst)
	rs, e1 := filepath.EvalSymlinks(src)
	rd, e2 := filepath.EvalSymlinks(dst)
	if e1 == nil && rs != src {
		if e2 != nil {
			rd = dst
		}
		add(rs, rd)
	}
	return from, to
}

// copyFixture copies src to dst, rewriting src's path to dst's in every
// file but git objects; git's trace files of the template are left out.
// token is the template folder's name: a copied file that still has it
// names the template, which a test must never touch.
func copyFixture(t *testing.T, src, dst, token string) {
	t.Helper()
	from, to := pathPairs(src, dst)
	e := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			if d.Name() == "trace2" {
				// The copy's own, empty: the template's traces are not its.
				if e := os.MkdirAll(out, 0o700); e != nil {
					return e
				}
				return filepath.SkipDir
			}
			return os.MkdirAll(out, 0o755)
		}
		if !d.Type().IsRegular() {
			t.Fatalf("fixture %s: %s is not a regular file", src, rel)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if !strings.Contains(filepath.ToSlash(rel), "/objects/") {
			if bytes.IndexByte(b, 0) >= 0 {
				// A binary file (the board cache, git's index) cannot take
				// a path of another length: it must not hold one.
				for _, f := range from {
					if bytes.Contains(b, []byte(f)) {
						t.Fatalf("fixture %s: binary file %s names the template folder", src, rel)
					}
				}
			}
			for i := range from {
				b = bytes.ReplaceAll(b, []byte(from[i]), []byte(to[i]))
			}
			if bytes.Contains(b, []byte(token)) {
				t.Fatalf("fixture copy %s still names the template folder %s:\n%s", out, token, b)
			}
		}
		return os.WriteFile(out, b, info.Mode().Perm())
	})
	if e != nil {
		t.Fatal(e)
	}
}

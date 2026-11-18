package gitbranch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func identityTranslator() Translator {
	return Translator{
		Names: func(paths []string) map[string]string {
			out := map[string]string{}
			for _, p := range paths {
				out[p] = p
			}
			return out
		},
		Content: func(_, _ string, b []byte) []byte { return b },
	}
}

func TestReplayTreeRejectsCollidingTranslations(t *testing.T) {
	root, _ := fixture(t, false)
	r := openRepo(t, root, "huginn", ".huginn")
	writeFile(t, r, "a.md", "first")
	writeFile(t, r, "b.md", "second")
	must(t, r.Commit(nil, "two files"))
	tr := identityTranslator()
	tr.Names = func(paths []string) map[string]string {
		return map[string]string{"a.md": "same.md", "b.md": "same.md"}
	}
	_, e := r.tree("HEAD", tr)
	if e == nil || !strings.Contains(e.Error(), "same.md") || !strings.Contains(e.Error(), "a.md") || !strings.Contains(e.Error(), "b.md") {
		t.Fatalf("colliding files must not overwrite one another: %v", e)
	}
}

func TestReplayTreeRejectsUnparseableMode(t *testing.T) {
	root, _ := fixture(t, false)
	r := openRepo(t, root, "huginn", ".huginn")
	writeFile(t, r, "link", "/outside/the/board")
	must(t, r.Commit(nil, "file"))
	blob := command(t, r.Dir, "rev-parse", "HEAD:link")
	command(t, r.Dir, "update-index", "--cacheinfo", "120000,"+blob+",link")
	command(t, r.Dir, "commit", "-qm", "link")
	_, e := r.tree("HEAD", identityTranslator())
	if e == nil || !strings.Contains(e.Error(), "link") || !strings.Contains(e.Error(), "120000") {
		t.Fatalf("symlink must not be read or written as a regular file: %v", e)
	}
	// No symlink privilege is required on Windows: only the index and
	// commit contain a symlink. The working tree remains an ordinary file.
	if info, e := os.Lstat(filepath.Join(r.Dir, "link")); e != nil || info.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("fixture unexpectedly installed a symlink: %v %v", info, e)
	}
}

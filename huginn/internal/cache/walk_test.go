package cache

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// 44w1ev check 3 (a), (b): the ReadDir walk with parallel stats finds the
// same task files with the same markers as filepath.WalkDir did, on a tree
// with groups, nested groups, task files folders (which hold .md files that
// are not tasks), hidden folders, a folder named like an id without a task,
// non-Markdown files and a symlink.
func TestWalkMarkersMatchesWalkDir(t *testing.T) {
	root := t.TempDir()
	files := []string{
		"a.aaaaaa.top.md", "aaaaaa/notes.md", "aaaaaa/deep/x.md",
		"bbbbbb.norank.md", "bbbbbb/log.md",
		"api/c.cccccc.api.md", "api/cccccc/r.md", "api/sub/d.dddddd.sub.md",
		"eeeeee/orphan.md", // an id-named folder with no task beside it: walked
		".hidden/f.ffffff.h.md", "api/.cache/x.md",
		"readme.txt", "api/g.gggggg.g.md.bak",
	}
	for _, f := range files {
		put(t, root, f, []byte("x "+f))
	}
	if e := os.Symlink(filepath.Join(root, "a.aaaaaa.top.md"), filepath.Join(root, "link.md")); e != nil {
		t.Fatal(e)
	}
	want := map[string]Marker{}
	e := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			if rel, _ := filepath.Rel(root, path); path != root && task.IsTaskFolder(root, filepath.ToSlash(rel)) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		st, _ := d.Info()
		rel, _ := filepath.Rel(root, path)
		want[filepath.ToSlash(rel)] = Marker{Path: filepath.ToSlash(rel), MtimeNS: st.ModTime().UnixNano(), Size: st.Size()}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
	got, e := walkMarkers(context.Background(), root, 0)
	if e != nil {
		t.Fatal(e)
	}
	if len(got) != len(want) {
		t.Fatalf("walk found %d files, WalkDir %d:\n got %v\nwant %v", len(got), len(want), got, want)
	}
	for p, m := range want {
		if got[p] != m {
			t.Errorf("%s: got %+v, want %+v", p, got[p], m)
		}
	}
	for _, p := range []string{"a.aaaaaa.top.md", "bbbbbb.norank.md", "api/c.cccccc.api.md", "api/sub/d.dddddd.sub.md", "eeeeee/orphan.md"} {
		if _, ok := got[p]; !ok {
			t.Errorf("missing %s", p)
		}
	}
}

// A sync records how it learned what changed: a full sync walks, a write's
// own refresh names its files.
func TestSyncRecordsPath(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	put(t, root, "a.aaaaaa.one.md", fixture(1))
	c := openTest(t, root, filepath.Join(t.TempDir(), "cache.db"), &fakeGit{head: "h"})
	defer c.Close()
	c.mu.Lock()
	st, e := c.sync(ctx, nil)
	c.mu.Unlock()
	if e != nil || st.Path != PathWalk {
		t.Fatalf("full sync: %+v %v", st, e)
	}
	c.mu.Lock()
	st, e = c.sync(ctx, map[string]bool{"a.aaaaaa.one.md": true})
	c.mu.Unlock()
	if e != nil || st.Path != PathOwnWrite {
		t.Fatalf("own write: %+v %v", st, e)
	}
}

package task

import (
	"os"
	"path/filepath"
	"testing"
)

// FolderOwners decides from one listing what IsTaskFolder decides with two
// globs per folder (np2kk2): same answer for every folder in a listing.
func TestFolderOwnersMatchesIsTaskFolder(t *testing.T) {
	root := t.TempDir()
	files := []string{
		"0a.aaaaaa.rank-and-slug.md", // *.<id>.*.md
		"bbbbbb.slug-only.md",        // <id>.*.md
		"zz.cccccc.x.y.md",           // id in the middle of more parts
		"dddddd.md",                  // too few parts: not an owner
		"x.eeeeee.md",                // id second of two parts before .md: not an owner
		"ffffff.notes.txt",           // not Markdown
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	dirs := []string{"aaaaaa", "bbbbbb", "cccccc", "dddddd", "eeeeee", "ffffff", "gggggg", "x", "zz", "0a", "rank-and-slug", "slug-only"}
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	owners := FolderOwners(entries)
	for _, d := range dirs {
		if got, want := owners[d], IsTaskFolder(root, d); got != want {
			t.Errorf("%s: FolderOwners %v, IsTaskFolder %v", d, got, want)
		}
	}
	for _, d := range []string{"aaaaaa", "bbbbbb", "cccccc"} {
		if !owners[d] {
			t.Errorf("%s is a files folder", d)
		}
	}
}

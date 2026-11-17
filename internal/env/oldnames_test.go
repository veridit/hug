package env

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// m1c28r checks 1 and 4: no file of the repository names an old variable
// (HUGINN_*, MUNINN_*) but the changelogs, which record history. A name
// spelt in parts ("HUG" + "INN_USER") is how a test sets an old name on
// purpose, and does not match. The old golden-update variable came back
// in a TUI test (e0af424) after the rename: this test keeps it out.
func TestNoOldNamesInRepository(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root not at %s: %v", root, err)
	}
	old := regexp.MustCompile(`\b(?:HUG` + `INN|MUN` + `INN)_[A-Z0-9_]+`)
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata" || name == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if name == "CHANGELOG.md" || !textFile(name) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			if m := old.FindString(line); m != "" {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s:%d names %s; the variable is HUG_%s", filepath.ToSlash(rel), i+1, m, m[strings.IndexByte(m, '_')+1:])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func textFile(name string) bool {
	switch filepath.Ext(name) {
	case ".go", ".md", ".sh", ".py", ".yml", ".yaml", ".toml", ".json", ".txt", ".rb":
		return true
	}
	return false
}

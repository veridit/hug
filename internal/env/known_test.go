package env

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// a0p5kt check 2: every HUG_ name in the repository (code, scripts, CI,
// docs) is on Known, so hug never refuses a variable it or its tests use,
// and a new one cannot be added without its line (and, if a person sets
// it, its hug help line). The changelog records history and is skipped;
// testdata goldens show paths, not settings. A name a test builds on
// purpose to be unknown is spelt in parts ("HUG_" + "NO_SYNC").
func TestEveryHUGNameIsKnown(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("module root not at %s: %v", root, err)
	}
	name := regexp.MustCompile(`\bHUG_[A-Z][A-Z0-9_]*[A-Z0-9]\b`)
	found := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		base := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(base, ".") && base != ".github" || base == "testdata" || base == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if base == "CHANGELOG.md" || !textFile(base) && base != "install.sh" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(b), "\n") {
			for _, m := range name.FindAllString(line, -1) {
				found++
				if IsKnown(m) {
					continue
				}
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s:%d uses %s, which is not on env.Known: add it there (hug refuses unknown HUG_ variables)", filepath.ToSlash(rel), i+1, m)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if found < 50 {
		t.Fatalf("scan found only %d HUG_ names: is the root right (%s)?", found, root)
	}
}

// The known list has no duplicates and every user-set variable says what
// it does, for hug help.
func TestKnownList(t *testing.T) {
	seen := map[string]bool{}
	for _, v := range Known {
		if seen[v.Name] {
			t.Errorf("%s twice", v.Name)
		}
		seen[v.Name] = true
		if !strings.HasPrefix(v.Name, "HUG_") {
			t.Errorf("%s is not a HUG_ name", v.Name)
		}
		if v.Kind == UserSet && (v.Help == "" || v.Example == "") {
			t.Errorf("%s: user-set variables need a help line and an example value", v.Name)
		}
	}
}

package recall

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// build_fresh.py builds the fresh sets recall changes are judged on; its
// NOTICE filter must leave out every message recall skips as a harness
// notice, or a later fresh set would score recall on messages it never
// answers (w4qmr1). The script lists each prefix as a Python string.
func TestBuildFreshFiltersEveryHarnessNotice(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "scripts", "build_fresh.py"))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range harnessNotices {
		if !containsQuoted(string(b), p) {
			t.Errorf("muninn/scripts/build_fresh.py NOTICE lacks %q", p)
		}
	}
}

// containsQuoted reports whether src holds s as a double-quoted literal.
func containsQuoted(src, s string) bool { return strings.Contains(src, strconv.Quote(s)) }

package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// txjkrm check 2: hug.toml tags = [...] is read; lint flags an open
// task's tag that is not on the list, with a fix: line that runs; after
// it lint is clean. hug task tags lists the board's tags.
func TestLintUnlistedTag(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--label", "perf")["task"].(Reply)["id"].(string)
	// No list yet: nothing judged.
	if _, out := runText(a, "lint"); !strings.HasPrefix(out, "no findings") {
		t.Fatalf("lint without a tags list:\n%s", out)
	}
	execute(t, a, "tags", "add", "compiler")
	toml, _ := os.ReadFile(filepath.Join(a.Cwd, TaskDir, task.ConfigName))
	if !strings.Contains(string(toml), "\ntags = [\"compiler\"]\n") {
		t.Fatalf("hug.toml:\n%s", toml)
	}
	// Not a variant: two ways out, add it or remove it.
	code, out := runText(a, "lint", id)
	if !strings.Contains(out, "[unlisted-tag] unlisted tag: perf is not in hug.toml tags\n") ||
		!strings.Contains(out, "  or: hug task set "+id+" --tag perf=\n") || !strings.HasSuffix(out, "next: hug task tags add perf\n") {
		t.Fatalf("lint %s (exit %d):\n%s", id, code, out)
	}
	// A variant: one way out, naming the listed tag; it runs, twice.
	execute(t, a, "set", id, "--tag", "Compiler")
	if _, out = runText(a, "lint", id); !strings.Contains(out, "unlisted tag: Compiler is a variant of compiler, which hug.toml tags lists") ||
		!strings.Contains(out, "hug task set "+id+" --tag Compiler=compiler") {
		t.Fatalf("variant:\n%s", out)
	}
	execute(t, a, "set", id, "--tag", "Compiler=compiler")
	if r := execute(t, a, "set", id, "--tag", "Compiler=compiler"); r["changed"] != false {
		t.Fatalf("second replace changed: %v", r["changed"])
	}
	if f := taskFile(t, a, id); !strings.Contains(f, `labels: ["perf","compiler"]`) {
		t.Fatalf("labels after the replace:\n%s", f)
	}
	if code, out = runText(a, "tags", "add", "perf"); code != 0 {
		t.Fatalf("tags add: %d\n%s", code, out)
	}
	if _, out = runText(a, "lint"); !strings.HasPrefix(out, "no findings") {
		t.Fatalf("lint after the fix:\n%s", out)
	}
	if r := execute(t, a, "tags"); strings.Join(toStrings(r["tags"]), ",") != "compiler,perf" {
		t.Fatalf("tags: %v", r["tags"])
	}
	// A bad name is refused with a fix.
	if code, out = runText(a, "tags", "add", "two words"); code != 2 || !strings.Contains(out, "fix: hug task tags add two-words") {
		t.Fatalf("bad name: %d\n%s", code, out)
	}
}

func toStrings(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		var out []string
		for _, s := range x {
			out = append(out, s.(string))
		}
		return out
	}
	return nil
}

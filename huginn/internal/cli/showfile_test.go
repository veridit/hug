package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// show --file prints the file in text mode (2026-11-16: it printed only the
// next hint; the content was reachable only with --json).
func TestShowFileText(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	r := execute(t, a, "new", "Files", "--goal", "g", "--accept", "a")
	id := r["task"].(Reply)["id"].(string)
	src := filepath.Join(t.TempDir(), "notes.md")
	os.WriteFile(src, []byte("bench results\nline 2\n"), 0o644)
	execute(t, a, "attach", id, src, "--name", "bench/notes.md")
	out := &bytes.Buffer{}
	a.Out = out
	if code := a.Run(context.Background(), []string{"show", id, "--file", "bench/notes.md"}); code != 0 {
		t.Fatal(code, a.Err)
	}
	if !strings.HasPrefix(out.String(), "bench results\nline 2\n") || !strings.Contains(out.String(), "next: ") {
		t.Fatalf("%q", out.String())
	}
}

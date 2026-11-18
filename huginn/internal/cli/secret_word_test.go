package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 4c7327: 'sk-' inside a word ('task-900-...') is not a key. A key starts at
// a word boundary; a real one is still refused (crab 00:39: attaching a
// seid report was refused over 'task-900-task979-ownership-...md').
func TestAttachSkInsideAWordIsNoSecret(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	r := execute(t, a, "new", "Files", "--goal", "g", "--purpose", "P", "--principle", "Q", "--accept", "a")
	id := r["task"].(Reply)["id"].(string)
	dir := t.TempDir()
	plain := filepath.Join(dir, "report.md")
	os.WriteFile(plain, []byte("see task-900-task979-ownership-reconciliation-2026-08-17.md and desk-12345678901234567890ab\n"), 0o644)
	execute(t, a, "attach", id, plain)
	// Built at run time: a literal fixture trips the repo's no-secrets gate.
	key := filepath.Join(dir, "key.md")
	os.WriteFile(key, []byte("OPENAI=sk-"+strings.Repeat("Ab3", 13)+"x\n"), 0o644)
	out := &bytes.Buffer{}
	a.Out, a.Err = out, out
	if code := a.Run(context.Background(), []string{"attach", id, key}); code == 0 || !strings.Contains(out.String(), "credential-shaped text near sk-") {
		t.Fatalf("a real key was not refused: exit %d\n%s", code, out)
	}
	for _, s := range []string{"(sk-" + strings.Repeat("Q", 24) + ")", "key:sk-ant-" + strings.Repeat("a1", 12), "\"sk-proj-" + strings.Repeat("Z9", 12) + "\""} {
		if secretHint([]byte(s)) == "" {
			t.Errorf("missed a key after punctuation: %s", s)
		}
	}
}

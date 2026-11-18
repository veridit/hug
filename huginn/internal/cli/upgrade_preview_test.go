package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// asr4t2 (Jørgen 2026-11-17 18:58): hug upgrade previews by default. A plain
// run changes nothing on disk, says per part what it would change, ends
// with fix: hug upgrade --apply and exits 5 (changes pending); --apply does
// the work and exits 0; with nothing left, a plain run exits 0. hug task
// upgrade is the same command.
func TestUpgradePreviewsThenApplies(t *testing.T) {
	a := app(t)
	a.Env["HOME"] = t.TempDir()
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "wait", id, "jhf: look at the colours", "--as", "nautilus")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no")
	board := filepath.Join(a.Cwd, TaskDir)
	head := git(t, board, "rev-parse", "HEAD")
	m, _ := globID(board, id)
	before, _ := os.ReadFile(m[0])

	code, out := runText(a, "upgrade")
	if code != 5 || !strings.Contains(out, "tasks: would: 1 questions to a person moved from waiting to asked ("+id+")") || !strings.HasSuffix(out, "fix: hug upgrade --apply\n") {
		t.Fatalf("plain upgrade: exit %d, want 5 ending fix: hug upgrade --apply:\n%s", code, out)
	}
	if after, _ := os.ReadFile(m[0]); string(after) != string(before) || git(t, board, "rev-parse", "HEAD") != head || git(t, board, "status", "--porcelain") != "" {
		t.Fatalf("a plain upgrade changed the board:\n%s", after)
	}
	if code, out = runText(a, "upgrade", "--dry-run"); code != 2 {
		t.Fatalf("--dry-run is gone (the preview is the default): exit %d\n%s", code, out)
	}

	code, out = runText(a, "upgrade", "--apply")
	if code != 0 || !strings.Contains(out, "tasks: 1 questions to a person moved from waiting to asked ("+id+")") || strings.Contains(out, "would:") {
		t.Fatalf("upgrade --apply: exit %d:\n%s", code, out)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, `asked_for: "jhf"`) || !strings.Contains(f, `asked_what: "look at the colours"`) || !strings.Contains(f, `stage: "asked"`) || strings.Contains(f, "\nwaiting_") {
		t.Fatalf("--apply did not move the wait:\n%s", f)
	}

	code, out = runText(a, "upgrade")
	if code != 0 || strings.Contains(out, "would:") || strings.Contains(out, "--apply") || !strings.Contains(out, "tasks: nothing to do") {
		t.Fatalf("nothing left: exit %d:\n%s", code, out)
	}
	if code, out = runText(a, "upgrade", "--apply"); code != 0 {
		t.Fatalf("--apply with nothing to do: exit %d:\n%s", code, out)
	}
}

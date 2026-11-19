package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// f5vx4x: a board's upgrade never changes this machine's files. In a clone
// with a fake HOME holding a hook config hug wrote with an old variable
// name, hug upgrade --apply moves the board and leaves the config
// byte-identical, never runs hug mem upgrade --apply, prints this machine's
// changes as pending under their own heading and exits 5 with fix: hug
// upgrade --apply --machine; that command then changes them (backup kept)
// and exits 0. On 2026-11-18 04:53 an upgrade in a scratch clone rewrote
// six live jcode configs.
func TestUpgradeTouchesMachineOnlyWithMachine(t *testing.T) {
	home := "MUN" + "INN_HOME"
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("JCODE_HOME", "")
	cfg := filepath.Join(h, ".jcode", "config.toml")
	if e := os.MkdirAll(filepath.Dir(cfg), 0o755); e != nil {
		t.Fatal(e)
	}
	old := "[hooks]\nturn_start = \"" + home + "=/m " + hugBin(t, h) + "/muninn hook jcode\"\n"
	if e := os.WriteFile(cfg, []byte(old), 0o644); e != nil {
		t.Fatal(e)
	}
	a := app(t)
	a.Env["HOME"], a.Env["JCODE_HOME"] = h, "" // this test's machine
	execute(t, a, "init")
	id := execute(t, a, "new", "Look", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	execute(t, a, "take", id, "--as", "nautilus", "--model", "m")
	execute(t, a, "wait", id, "jhf: look at the colours", "--as", "nautilus")
	execute(t, a, "people", "add", "jhf", "jorgen@veridit.no")
	var memCalls []string
	folded := false
	a.MemUpgrade = func(args []string) (int, string) {
		memCalls = append(memCalls, strings.Join(append([]string{"mem", "upgrade"}, args...), " "))
		if folded {
			return 0, "memory: nothing to do\n"
		}
		if len(args) == 1 && args[0] == "--apply" {
			folded = true
			return 0, "memory: jcode[default] recall now arrives in a tool result\n"
		}
		return 5, "memory: would deliver jcode[default] recall in a tool result\nfix: hug mem upgrade --apply\n"
	}
	board := filepath.Join(a.Cwd, TaskDir)
	unchanged := func(when string) {
		t.Helper()
		if b, _ := os.ReadFile(cfg); string(b) != old {
			t.Fatalf("%s changed this machine's config:\n%s", when, b)
		}
		if _, e := os.Stat(cfg + ".bak-hug-upgrade"); !os.IsNotExist(e) {
			t.Fatalf("%s wrote a machine backup: %v", when, e)
		}
		for _, c := range memCalls {
			if strings.Contains(c, "--apply") {
				t.Fatalf("%s ran %q", when, c)
			}
		}
	}

	// The preview: two headings, both steps named, nothing changed.
	code, out := runText(a, "upgrade")
	// The heading names the board as git resolves it: /private/var on
	// macOS, long names and forward slashes on Windows (main CI
	// 44237364634). The same folder, not the same string.
	shown := ""
	if _, rest, ok := strings.Cut(out, "this board ("); ok {
		shown, _, _ = strings.Cut(rest, "):\n")
	}
	if st1, e1 := os.Stat(shown); e1 != nil {
		t.Fatalf("preview: board heading names %q: %v\n%s", shown, e1, out)
	} else if st2, e2 := os.Stat(board); e2 != nil || !os.SameFile(st1, st2) {
		t.Fatalf("preview: board heading names %q, want %s (%v)\n%s", shown, board, e2, out)
	}
	for _, want := range []string{
		"this board (" + shown + "):\n  layout: nothing to do\n  tasks: would: 1 questions to a person moved from waiting to asked (" + id + ")\n",
		"this machine (hook and MCP configs in ~/.jcode, ~/.claude, ~/.codex; the memory store):\n  variables: would: variables renamed to HUG_* in " + cfg + "\n  names: would: hooks and MCP entries run hug in " + cfg + "\n",
		"  memory: would deliver jcode[default] recall in a tool result\n",
		"next: hug upgrade --apply\nthen: hug upgrade --apply --machine  (this machine, once you have read it)\nfix: hug upgrade --apply\n",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("preview: exit %d, lacks %q:\n%s", code, want, out)
		}
	}
	if code != UpgradePending {
		t.Fatalf("preview exit %d", code)
	}
	unchanged("hug upgrade")

	// --apply: the board only.
	code, out = runText(a, "upgrade", "--apply")
	if !strings.Contains(out, "  tasks: 1 questions to a person moved from waiting to asked ("+id+")\n") || !strings.Contains(out, "  variables: would: variables renamed to HUG_* in "+cfg+"\n") || !strings.HasSuffix(out, "next: hug upgrade --apply --machine\nfix: hug upgrade --apply --machine\n") {
		t.Fatalf("--apply: exit %d\n%s", code, out)
	}
	if code != UpgradePending {
		t.Fatalf("--apply left machine work pending but exit %d (crab 04:04: 5)", code)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, `stage: "asked"`) {
		t.Fatalf("--apply did not move the board:\n%s", f)
	}
	unchanged("hug upgrade --apply")
	if code, out = runText(a, "upgrade", "--machine"); code != 2 || !strings.Contains(out, "--machine goes with --apply") || !strings.HasSuffix(out, "fix: hug upgrade --apply --machine\n") {
		t.Fatalf("--machine alone: exit %d\n%s", code, out)
	}
	unchanged("hug upgrade --machine")

	// --apply --machine: this machine too, with its backup.
	code, out = runText(a, "upgrade", "--apply", "--machine")
	if code != 0 || !strings.Contains(out, "  variables: variables renamed to HUG_* in "+cfg+" (backup config.toml.bak-hug-upgrade)\n") || !strings.Contains(out, "  memory: jcode[default] recall now arrives in a tool result\n") {
		t.Fatalf("--apply --machine: exit %d\n%s", code, out)
	}
	if b, _ := os.ReadFile(cfg); !strings.Contains(string(b), "HUG_HOME=/m "+hugBin(t, h)+"/hug mem hook jcode") {
		t.Fatalf("config after --machine:\n%s", b)
	}
	if b, _ := os.ReadFile(cfg + ".bak-hug-upgrade"); string(b) != old {
		t.Fatalf("backup:\n%s", b)
	}
	if code, out = runText(a, "upgrade"); code != 0 || strings.Contains(out, "would:") || strings.Contains(out, "fix:") {
		t.Fatalf("after both: exit %d\n%s", code, out)
	}
}

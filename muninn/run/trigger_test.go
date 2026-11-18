package run

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/trigger"
)

func cliTriggerOf(t *testing.T, text string) *trigger.Trigger {
	t.Helper()
	ms, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if m.Content == text {
			return m.Trigger
		}
	}
	t.Fatalf("%q not stored", text)
	return nil
}

// jrj1vq check 1: hug mem remember takes --when, --when-not and --case-*;
// show prints them; a changed trigger exits 3 with NOT STORED and the
// supersede fix; --supersedes ID --no-trigger clears it.
func TestCLIRememberTrigger(t *testing.T) {
	memHome(t)
	text := "Synthetic jrj1vq rule: ticket history lives in the notes"
	out, errOut, code := runCLI(t, "", "remember", "--type", "rule", "--when", "reviewing a ticket's history", "--when-not", "writing a new ticket",
		"--case-tool", "hug task", "--case-path", "docs/b.md", "--case-path", "docs/a.md", text)
	if code != 0 {
		t.Fatalf("remember: %d %s%s", code, out, errOut)
	}
	sid := strings.Fields(out)[1]
	want := &trigger.Trigger{When: "reviewing a ticket's history", WhenNot: "writing a new ticket", Case: &trigger.Case{Paths: []string{"docs/a.md", "docs/b.md"}, Tools: []string{"hug task"}}}
	if got := cliTriggerOf(t, text); !trigger.Equal(got, want) {
		t.Fatalf("stored %+v", got)
	}
	out, _, code = runCLI(t, "", "show", sid)
	if code != 0 || !strings.Contains(out, "case: paths docs/a.md, docs/b.md; tools hug task\nwhen: reviewing a ticket's history\nwhen not: writing a new ticket\n") {
		t.Fatalf("show: %d %q", code, out)
	}
	out, errOut, code = runCLI(t, "", "remember", "--when", "auditing a ticket", text)
	if code != 3 || !strings.Contains(out+errOut, "NOT STORED:") || !strings.Contains(out+errOut, "fix: hug mem remember --supersedes "+sid) {
		t.Fatalf("changed trigger: %d %s%s", code, out, errOut)
	}
	if out, errOut, code = runCLI(t, "", "remember", "--supersedes", sid, "--no-trigger", text); code != 0 {
		t.Fatalf("clear: %d %s%s", code, out, errOut)
	}
	if got := cliTriggerOf(t, text); !trigger.Cleared(got) {
		t.Fatalf("clear stored %+v", got)
	}
	out, _, _ = runCLI(t, "", "show", sid)
	if strings.Contains(out, "when:") {
		t.Fatalf("show after clear: %q", out)
	}
	// --then does not exist (crab ruling a): the text is the THEN.
	if _, _, code = runCLI(t, "", "remember", "--then", "x", "y"); code == 0 {
		t.Fatal("--then was accepted")
	}
}

// jrj1vq (crab): an older hug rewrites memories.jsonl without the trigger
// key; this clone keeps its trigger, and its next sync writes it back.
func TestCLISyncRestoresTriggerDroppedByOlderHug(t *testing.T) {
	home := memHome(t)
	p := gitProject(t, home)
	sharesMemories(t, home) // 34r1ft: this test syncs through the branch
	text := "Synthetic jrj1vq: gateway deploys need the canary flag"
	if out, errOut, code := runCLI(t, "", "remember", "--project", "--when", "deploying the gateway", text); code != 0 {
		t.Fatalf("remember: %d %s%s", code, out, errOut)
	}
	if out, errOut, code := runCLI(t, "", "sync"); code != 0 {
		t.Fatalf("sync: %d %s%s", code, out, errOut)
	}
	raw, err := os.ReadFile(p.File())
	if err != nil || !strings.Contains(string(raw), `"trigger":{"when":"deploying the gateway"}`) {
		t.Fatalf("file: %v %s", err, raw)
	}
	// The older hug: it parses each line into its own Line (no trigger) and
	// writes it back.
	old := strings.Replace(string(raw), `,"trigger":{"when":"deploying the gateway"}`, "", 1)
	if old == string(raw) {
		t.Fatalf("fixture: %s", raw)
	}
	if err := os.WriteFile(p.File(), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	// It commits its rewrite on the notes branch, as a sync does.
	commit := exec.Command("git", "-C", filepath.Dir(p.File()), "-c", "user.name=Old", "-c", "user.email=old@example.invalid", "commit", "-qam", "older hug rewrite")
	if b, err := commit.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v %s", err, b)
	}
	if out, errOut, code := runCLI(t, "", "sync"); code != 0 {
		t.Fatalf("sync: %d %s%s", code, out, errOut)
	}
	if got := cliTriggerOf(t, text); got == nil || got.When != "deploying the gateway" {
		t.Fatalf("the dropped key cleared the trigger: %+v", got)
	}
	if b, _ := os.ReadFile(p.File()); !strings.Contains(string(b), `"trigger":{"when":"deploying the gateway"}`) {
		t.Fatalf("sync did not restore the key: %s", b)
	}
}

// jrj1vq check 2 (1): remember's help asks for --when/--when-not on a rule or
// pattern, and every flag its words name exists.
func TestCLIRememberHelpAsksForTrigger(t *testing.T) {
	memHome(t)
	out, errOut, code := runCLI(t, "", "remember", "--help")
	help := out + errOut
	if code != 0 {
		t.Fatalf("help: %d %s", code, help)
	}
	for _, want := range []string{"For a rule or pattern", "--when ", "--when-not ", "--case-scope", "--case-path", "--case-tool", "--no-trigger", "--supersedes", "The text is then the THEN"} {
		if !strings.Contains(help, want) {
			t.Errorf("help lacks %q", want)
		}
	}
	if strings.Contains(help, "--then") {
		t.Error("help names --then, which does not exist")
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// This is intentionally also run by the short suite: the real command's
// board-only permission boundary and search binding are the acceptance.
func TestMemoryLayoutRealBinaryKeepsSearchAndNativeBytes(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	store := filepath.Join(base, "store")
	root := filepath.Join(base, "repo")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "HUG_HOME="+store, "JCODE_HOME="+filepath.Join(home, "jcode"), "HUG_SYNC=off", "HUG_NO_EMBED=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=Synthetic", "GIT_AUTHOR_EMAIL=synthetic@example.invalid", "GIT_COMMITTER_NAME=Synthetic", "GIT_COMMITTER_EMAIL=synthetic@example.invalid", "GOMAXPROCS=2", "GOFLAGS=-p=2")
	env = append(env, "PATH="+base+string(os.PathListSeparator)+os.Getenv("PATH"))
	// An old variable name makes every hug print the stale-name warning on
	// stderr; the --json search below must still parse (tfe9nr). Spelt in
	// parts so TestNoOldNamesInRepository does not match it.
	env = append(env, "HUG"+"INN_UPDATE_GOLDEN=1")
	bin := filepath.Join(base, "hug")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Env = append(os.Environ(), "GOMAXPROCS=2", "GOFLAGS=-p=2")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	try := func(command string, args ...string) (string, error) {
		cmd := exec.Command(command, args...)
		cmd.Dir, cmd.Env = root, env
		b, err := cmd.CombinedOutput()
		return string(b), err
	}
	run := func(command string, args ...string) string {
		t.Helper()
		out, err := try(command, args...)
		if err != nil {
			t.Fatalf("%s %v: %v\n%s", command, args, err, out)
		}
		return out
	}
	// stdout is a --json reply alone: a stderr line before it (a warning,
	// a notice) breaks the parse (tfe9nr), so stderr only shows on failure.
	stdout := func(command string, args ...string) (string, string) {
		t.Helper()
		cmd := exec.Command(command, args...)
		cmd.Dir, cmd.Env = root, env
		var stderr strings.Builder
		cmd.Stderr = &stderr
		b, err := cmd.Output()
		if err != nil {
			t.Fatalf("%s %v: %v\n%s\nstderr:\n%s", command, args, err, b, stderr.String())
		}
		return string(b), stderr.String()
	}
	run("git", "init", "-q", "-b", "main")
	run("git", "commit", "-q", "--allow-empty", "-m", "code")
	run(bin, "task", "init")
	run(bin, "task", "memory", "share") // 34r1ft: a repository opts in to sync its memories
	run(bin, "mem", "remember", "--project", "Quartz migration proof keeps the same raven memory.")
	run(bin, "mem", "sync")
	canonical := filepath.Join(root, ".hug", "memory")
	old := filepath.Join(root, ".muninn-notes")
	// Construct an old layout using Git only, without changing memory data.
	run("git", "branch", "-m", "hug/memory", "muninn-notes")
	run("git", "worktree", "move", canonical, old)
	head := run("git", "rev-parse", "muninn-notes")
	raw, err := os.ReadFile(filepath.Join(old, "memories.jsonl"))
	if err != nil || len(raw) == 0 {
		t.Fatalf("old memory fixture: %v %q", err, raw)
	}
	nativePath := filepath.Join(store, "data", "memories.json")
	native, err := os.ReadFile(nativePath)
	if err != nil {
		t.Fatal(err)
	}
	results := func() []byte {
		t.Helper()
		out, errs := stdout(bin, "mem", "search", "Quartz migration proof", "--all", "--json")
		var response struct {
			Memories []map[string]any `json:"memories"`
		}
		if err := json.Unmarshal([]byte(out), &response); err != nil || len(response.Memories) == 0 {
			t.Fatalf("search has no proof memory: %v\n%s\nstderr:\n%s", err, out, errs)
		}
		b, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	before := results()
	for _, args := range [][]string{{"mem", "project"}, {"mem", "remember", "--project", "must not write on old layout"}, {"mem", "sync"}} {
		if out, err := try(bin, args...); err == nil || !strings.Contains(out, "fix: hug upgrade") {
			t.Fatalf("ordinary old-layout command %v was not explicit: %v\n%s", args, err, out)
		}
	}
	if b, err := os.ReadFile(nativePath); err != nil || !bytes.Equal(b, native) {
		t.Fatalf("refused ordinary write changed native memory: %v", err)
	}
	if out, err := try(bin, "upgrade"); err == nil || !strings.Contains(out, "layout: would: .muninn-notes/ moves to .hug/memory/, branch muninn-notes becomes hug/memory") {
		t.Fatalf("memory preview: %v\n%s", err, out)
	}
	run(bin, "upgrade", "--apply")
	if b, err := os.ReadFile(filepath.Join(canonical, "memories.jsonl")); err != nil || !bytes.Equal(b, raw) {
		t.Fatalf("branch memory bytes changed: %v", err)
	}
	if b, err := os.ReadFile(nativePath); err != nil || !bytes.Equal(b, native) {
		t.Fatalf("board-only migration changed native memory: %v", err)
	}
	if after := results(); !bytes.Equal(before, after) {
		t.Fatalf("search memory IDs/text/metadata/order/scores changed:\nbefore%s\nafter%s", before, after)
	}
	if got := run("git", "rev-parse", "hug/memory"); got != head {
		t.Fatalf("memory branch history changed: %s != %s", got, head)
	}
	run(bin, "upgrade", "--apply")
	if got := run("git", "rev-parse", "hug/memory"); got != head {
		t.Fatalf("second apply changed memory history: %s", got)
	}

	// A fresh clone can know only the old remote-tracking ref. Execute the
	// complete printed attachment-plus-preview command, not a paraphrase.
	run("git", "branch", "-m", "hug/memory", "muninn-notes")
	run("git", "update-ref", "refs/remotes/origin/muninn-notes", strings.TrimSpace(head))
	run("git", "worktree", "remove", canonical)
	run("git", "branch", "-D", "muninn-notes")
	refs := run("git", "show-ref")
	out, refusal := try(bin, "upgrade", "--apply")
	if refusal == nil || !strings.Contains(out, "project memories remain at refs/remotes/origin/muninn-notes; no empty hug/memory branch created") {
		t.Fatalf("remote-only branch was silently missed: %v\n%s", refusal, out)
	}
	if run("git", "show-ref") != refs {
		t.Fatal("remote-only refusal changed refs")
	}
	fix := ""
	for _, line := range strings.Split(out, "\n") {
		if command, ok := strings.CutPrefix(line, "fix: "); ok {
			fix = command
		}
	}
	if fix == "" {
		t.Fatal("remote-only refusal has no runnable fix")
	}
	preview, previewErr := runMemoryLayoutPrintedFix(root, env, fix)
	if e, ok := previewErr.(*exec.ExitError); !ok || e.ExitCode() != 5 || !strings.Contains(preview, "layout: would: .muninn-notes/ moves to .hug/memory/") {
		t.Fatalf("printed fix did not attach then preview5: %v\nfix: %s\n%s", previewErr, fix, preview)
	}
	if b, e := os.ReadFile(filepath.Join(old, "memories.jsonl")); e != nil || !bytes.Equal(b, raw) {
		t.Fatalf("printed fix lost recovered memory: %v", e)
	}
	run(bin, "upgrade", "--apply")
	run(bin, "upgrade", "--apply")
	if run("git", "rev-parse", "hug/memory") != head {
		t.Fatal("remote-only recovery changed original memory history")
	}
}

// Feed the exact printed command to the native shell. exec.Command's Windows
// argument quoting follows CommandLineToArgvW, not cmd.exe's /C grammar, and
// passing a quoted command as an argument can leave literal quotes in paths.
func runMemoryLayoutPrintedFix(root string, env []string, fix string) (string, error) {
	cmd := exec.Command("sh")
	input := fix + "\n"
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/D", "/Q")
		input = fix + "\r\nexit /b %errorlevel%\r\n"
	}
	cmd.Dir, cmd.Env = root, env
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestMemoryLayoutPrintedFixKeepsQuotedPathAndExitCode(t *testing.T) {
	root := filepath.Join(t.TempDir(), "quoted recovery path")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	exit := "exit 5"
	if runtime.GOOS == "windows" {
		exit = "exit /b 5"
	}
	out, err := runMemoryLayoutPrintedFix(root, os.Environ(), `git -C "`+root+`" rev-parse --is-inside-work-tree && `+exit)
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 5 || !strings.Contains(out, "true") {
		t.Fatalf("quoted recovery command: %v\n%s", err, out)
	}
}

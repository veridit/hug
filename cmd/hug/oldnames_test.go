package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// p277bp check 2: users type only hug. No help text tells anyone to type
// huginn or muninn, or calls the tool by those names: the help of every
// command, found by walking the command tree as a user would (hug help,
// hug task help and each command it lists, hug mem --help and each
// command it lists, recursively). The parts keep their names as folders
// and files (huginn/internal/..., .muninn/, muninn-notes), which is not a
// word on its own.
func TestNoHelpNamesTheOldCommands(t *testing.T) {
	t.Setenv("HUG_SYNC", "off")
	t.Setenv("HUG_HOME", t.TempDir())
	t.Setenv("HUG_NO_EMBED", "1")
	t.Chdir(t.TempDir())
	help := func(args ...string) string {
		var errOut string
		out := capture(t, &os.Stdout, func() {
			errOut = capture(t, &os.Stderr, func() { run("hug", args) })
		})
		return out + errOut
	}
	texts := map[string]string{}
	for _, args := range [][]string{{"help"}, {"mcp", "--help"}, {"apps", "--help"}, {"completion", "--help"}, {"task", "help"}} {
		texts["hug "+strings.Join(args, " ")] = help(args...)
	}
	// hug task: every command hug task help lists.
	n := 0
	for _, line := range strings.Split(texts["hug task help"], "\n") {
		if _, list, ok := strings.Cut(line, ":"); ok && taskGroup.MatchString(line) {
			for _, c := range strings.Fields(list) {
				texts["hug task help "+c] = help("task", "help", c)
				n++
			}
		}
	}
	if n < 40 {
		t.Fatalf("walked %d hug task commands, want every one hug task help lists:\n%s", n, texts["hug task help"])
	}
	// hug mem: every command its help lists, and theirs.
	var walk func(path []string)
	walk = func(path []string) {
		args := append(append([]string{"mem"}, path...), "--help")
		out := help(args...)
		texts["hug "+strings.Join(args, " ")] = out
		for _, c := range cobraCommands(out) {
			if c != "help" {
				walk(append(append([]string(nil), path...), c))
			}
		}
	}
	walk(nil)
	if len(cobraCommands(texts["hug mem --help"])) < 20 {
		t.Fatalf("hug mem --help lists too few commands to walk:\n%s", texts["hug mem --help"])
	}
	bad := 0
	for name, text := range texts {
		for i, line := range strings.Split(text, "\n") {
			if oldName.MatchString(line) && !allowedOldName(line) {
				t.Errorf("%s line %d names an old command: %s", name, i+1, line)
				bad++
			}
		}
	}
	t.Logf("%d help texts scanned, %d lines name huginn or muninn", len(texts), bad)
}

// taskGroup is a line of hug task help's command list ("  work:   next take ...").
var taskGroup = regexp.MustCompile(`^  [a-z]+:\s+[a-z]`)

// cobraCommands are the commands a cobra help lists under Available Commands.
func cobraCommands(help string) []string {
	_, rest, ok := strings.Cut(help, "Available Commands:\n")
	if !ok {
		return nil
	}
	var out []string
	for _, line := range strings.Split(rest, "\n") {
		f := strings.Fields(line)
		if !strings.HasPrefix(line, "  ") || len(f) == 0 {
			break
		}
		out = append(out, f[0])
	}
	return out
}

// oldName is huginn or muninn as a word: not part of a path, a file or a
// branch name (huginn/internal/..., .muninn/, muninn-notes, huginn.toml).
var oldName = regexp.MustCompile(`(?i)(^|[^\w./-])(huginn|muninn)($|[^\w/.-])`)

// allowedOldName is a line that names an old layout as a fact of a
// migration, never as something to type.
func allowedOldName(line string) bool {
	for _, ok := range []string{
		"the old huginn branch and .huginn/", // hug upgrade --help: what the layout migration moves
	} {
		if strings.Contains(line, ok) {
			return true
		}
	}
	return false
}

// p277bp check 2 (crab 22:21): hug mem completion is gone (it completed
// the command muninn), and a completion script installed before still
// gets answers: cobra keeps its hidden __complete command.
func TestCompletionGoneInstalledScriptsAnswer(t *testing.T) {
	t.Setenv("HUG_SYNC", "off")
	t.Setenv("HUG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	var code int
	out := capture(t, &os.Stdout, func() { code = run("muninn", []string{"__complete", "se"}) })
	if code != 0 || !strings.Contains(out, "search\t") {
		t.Errorf("muninn __complete se: exit %d\n%s", code, out)
	}
	capture(t, &os.Stderr, func() {
		capture(t, &os.Stdout, func() { code = run("hug", []string{"mem", "completion", "zsh"}) })
	})
	if code == 0 {
		t.Error("hug mem completion zsh still writes a script for the command muninn")
	}
}

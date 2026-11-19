package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// 5mmqjk: hug completion bash|zsh|fish prints one script for the command
// hug, and each shell, given a half-typed line, offers the words hug's own
// trees know: its commands, hug task's (commands, flags, open task ids)
// and hug mem's (cobra's tree). The scripts run in the real shells, with
// the real binary on PATH. A shell that is not installed skips, unless
// HUG_REQUIRE_SHELLS=1 (CI's ubuntu job installs all three).
func TestCompletionInEachShell(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the hug binary")
	}
	if runtime.GOOS == "windows" {
		t.Skip("bash, zsh and fish scripts; Windows has none of them")
	}
	base := t.TempDir()
	binDir := filepath.Join(base, "bin")
	bin := filepath.Join(binDir, "hug")
	buildHug(t, bin, "dev")
	repo := filepath.Join(base, "repo")
	home := filepath.Join(base, "home")
	env := []string{"PATH=" + binDir + string(os.PathListSeparator) + os.Getenv("PATH"), "HOME=" + home, "HUG_HOME=" + filepath.Join(home, ".hug"),
		"HUG_SYNC=off", "HUG_NO_EMBED=1", "HUG_USER=tester", "HUG_MODEL=test-model", "JCODE_HOME=" + filepath.Join(home, "no-jcode"),
		"GIT_CONFIG_GLOBAL=" + os.DevNull, "GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.invalid",
		"GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.invalid", "TERM=dumb", "LANG=C"}
	sh := func(dir string, args ...string) string {
		t.Helper()
		// exec resolves a bare name on the test's own PATH: hug is the
		// one just built, never one installed on the machine.
		if args[0] == "hug" {
			args[0] = bin
		}
		c := exec.Command(args[0], args[1:]...)
		c.Dir, c.Env = dir, env
		out, err := c.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	os.MkdirAll(repo, 0o755)
	os.MkdirAll(home, 0o755)
	sh(repo, "git", "init", "-q", "-b", "main")
	sh(repo, "git", "commit", "-q", "--allow-empty", "-m", "init")
	sh(repo, "hug", "task", "init")
	id := strings.TrimSpace(sh(repo, "sh", "-c", `hug task new "Completion target" --goal g --purpose p --principle q --accept one --json | sed -n 's/.*"id": *"\([a-z0-9]*\)".*/\1/p' | head -1`))
	if got := strings.TrimSpace(sh(repo, "sh", "-c", "command -v hug")); got != bin {
		t.Fatalf("the shells find hug at %s, want %s", got, bin)
	}
	if len(id) != 6 {
		t.Fatalf("new task id %q", id)
	}
	scripts := map[string]string{}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		p := filepath.Join(base, "hug."+shell)
		if e := os.WriteFile(p, []byte(sh(repo, "hug", "completion", shell)), 0o644); e != nil {
			t.Fatal(e)
		}
		scripts[shell] = p
	}
	cases := []struct{ line, want, not string }{
		{"hug ta", "task", "tui"},
		{"hug task sh", "show", "take"},
		{"hug mem rec", "recall", "remember"},
		{"hug task ta", "take", "show"}, // 5mmqjk check 1
		{"hug mem se", "search", "recall"},
		{"hug task take --mo", "--model", "--goal"},
		{"hug task show " + id[:2], id, ""},
		{"hug task ls --state d", "doing", "todo"},
	}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		t.Run(shell, func(t *testing.T) {
			run, ok := shellDriver(t, shell, scripts[shell])
			if !ok {
				return
			}
			for _, tc := range cases {
				c := exec.Command(run[0], append(run[1:], tc.line)...)
				c.Dir, c.Env = repo, env
				out, err := c.CombinedOutput()
				if err != nil {
					t.Errorf("%s %q: %v\n%s", shell, tc.line, err, out)
					continue
				}
				got := strings.Fields(string(out))
				if !slices.Contains(got, tc.want) || (tc.not != "" && slices.Contains(got, tc.not)) {
					t.Errorf("%s %q: got %q, want %q and not %q", shell, tc.line, got, tc.want, tc.not)
				}
			}
		})
	}
}

// shellDriver is the command that completes a line (its last argument) in
// shell with the script sourced, printing one word per line; ok false when
// the shell is missing and the test may skip.
func shellDriver(t *testing.T, shell, script string) ([]string, bool) {
	t.Helper()
	path, err := exec.LookPath(shell)
	if err == nil && shell == "bash" {
		// The script needs bash 4.4 or newer (macOS ships 3.2).
		out, _ := exec.Command(path, "-c", `echo "${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"`).Output()
		if v := strings.TrimSpace(string(out)); v < "4.4" && !strings.HasPrefix(v, "5") {
			for _, p := range []string{"/opt/homebrew/bin/bash", "/usr/local/bin/bash"} {
				if _, e := os.Stat(p); e == nil {
					path = p
				}
			}
		}
	}
	if err != nil {
		if os.Getenv("HUG_REQUIRE_SHELLS") == "1" {
			t.Fatalf("%s is not installed and HUG_REQUIRE_SHELLS=1", shell)
		}
		t.Skipf("%s is not installed", shell)
		return nil, false
	}
	switch shell {
	case "bash":
		// COMP_WORDS as bash splits the line; the script's own
		// _get_comp_words_by_ref stand-in reads them when bash-completion
		// is not loaded.
		return []string{path, "-c", `source "$0"
line=$1
read -ra COMP_WORDS <<< "$line"
[[ $line == *" " ]] && COMP_WORDS+=("")
COMP_CWORD=$(( ${#COMP_WORDS[@]} - 1 ))
COMP_LINE=$line COMP_POINT=${#line}
_get_comp_words_by_ref() { cur=${COMP_WORDS[COMP_CWORD]}; prev=${COMP_WORDS[COMP_CWORD-1]}; words=("${COMP_WORDS[@]}"); cword=$COMP_CWORD; }
__start_hug
printf '%s\n' "${COMPREPLY[@]}"`, script}, true
	case "zsh":
		// _describe and compadd stand-ins print what the script offers.
		return []string{path, "-f", "-c", `compdef() { :; }
_describe() { local name=$2; shift 2; local -a items; items=("${(@P)name}"); for i in "${items[@]}"; do print -r -- "${i%%:*}"; done; }
_arguments() { :; }
compadd() { :; }
source "$0"
words=(${=1}); [[ $1 == *" " ]] && words+=("")
CURRENT=${#words}
_hug`, script}, true
	case "fish":
		// string replace exits 1 when no line had a description (hug
		// task's words have none): the words, not its status, are checked.
		return []string{path, "--no-config", "-c", `source $argv[1]; complete -C"$argv[2]" | string replace -r '\t.*' ''; true`, script}, true
	}
	return nil, false
}

// 5mmqjk: hug __complete dispatch. The first word comes from hug's own
// list; mem words from hug mem's cobra tree; the huginn link completes
// hug task and the muninn link hug mem (scripts installed before). A
// shell hug has no script for is refused with a fix: line.
func TestCompleteDispatch(t *testing.T) {
	t.Setenv("HUG_SYNC", "off")
	t.Setenv("HUG_HOME", t.TempDir())
	t.Setenv("HUG_NO_EMBED", "1")
	t.Chdir(t.TempDir())
	for _, tc := range []struct {
		name string
		args []string
		want string
		not  string
	}{
		{"hug", []string{"__complete", "ta"}, "task\t", "tui"},
		{"hug", []string{"__completeNoDesc", "ta"}, "task\n", "\t"},
		{"hug", []string{"__complete", "mem", "rec"}, "recall\t", "remember"},
		{"hug", []string{"__complete", "task", "sh"}, "show\n", "take"},
		{"hug", []string{"__complete", "completion", ""}, "fish\n", "powershell"},
		{"hug", []string{"__complete", "apps", ""}, "install\n", "show"},
		{"huginn", []string{"__complete", "sh"}, "show\n", "search"},
		{"muninn", []string{"__complete", "se"}, "search\t", "show"},
	} {
		var code int
		out := capture(t, &os.Stdout, func() { code = run(tc.name, tc.args) })
		if code != 0 || !strings.Contains(out, tc.want) || strings.Contains(out, tc.not) || !strings.Contains(out, "\n:4\n") {
			t.Errorf("%s %q: exit %d\n%s", tc.name, tc.args, code, out)
		}
	}
	var code int
	errOut := capture(t, &os.Stderr, func() { code = run("hug", []string{"completion", "powershell"}) })
	if code != 2 || !strings.Contains(errOut, "\nfix: hug completion zsh") {
		t.Errorf("hug completion powershell: exit %d\n%s", code, errOut)
	}
}

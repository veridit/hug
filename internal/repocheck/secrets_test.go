package repocheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 4c7327: the CI secrets job (muninn/scripts/check-secrets.sh, the same
// rule as hug attach and hug mem remember) passes 'sk-' inside a word and
// refuses a real key. Run on a throwaway repository holding one file.
func TestCheckSecretsWordBoundary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("check-secrets.sh runs on the ubuntu CI job")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "muninn", "scripts", "check-secrets.sh"))
	if err != nil {
		t.Fatal(err)
	}
	run := func(content string) (int, string) {
		t.Helper()
		dir := t.TempDir()
		for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"}} {
			if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("git", "-C", dir, "add", "notes.md").CombinedOutput(); err != nil {
			t.Fatalf("git add: %v\n%s", err, out)
		}
		cmd := exec.Command("bash", script, "--all")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH=/usr/bin:/bin") // no gitleaks: the script's own pattern decides
		out, _ := cmd.CombinedOutput()
		return cmd.ProcessState.ExitCode(), string(out)
	}
	if code, out := run("see task-900-task979-ownership-reconciliation-2026-08-17.md\n"); code != 0 {
		t.Fatalf("'task-900-...' refused (exit %d):\n%s", code, out)
	}
	// Built at run time: a literal key would trip this repository's own gate.
	if code, out := run("OPENAI=sk-" + strings.Repeat("Ab3", 13) + "x\n"); code == 0 || !strings.Contains(out, "secret-like content in notes.md") {
		t.Fatalf("a real key passed (exit %d):\n%s", code, out)
	}
}

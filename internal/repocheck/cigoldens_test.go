package repocheck

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// a0gwvg: scripts/ci-goldens.sh turns a failed CI run's 'golden drift'
// blocks back into golden files, byte for byte, and writes nothing when the
// tree's goldens are not the ones that run compared against.
//
// testdata/ci-goldens/failed.log is two blocks of CI 44403710185
// (test (ubuntu-latest), stack 6448e7d5) as gh run view --log-failed prints
// them; want/ holds that commit's palette goldens, got/ the ones 4d7eec2d
// committed from the same run. The 140x40 screen has 6 blank lines, and
// each block's last line is the screen's last row with trailing spaces.
func TestCIGoldensRebuildsDriftedGoldens(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ci-goldens.sh is a bash tool for macOS and Linux")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "scripts", "ci-goldens.sh"))
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := filepath.Abs(filepath.Join("testdata", "ci-goldens"))
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) []byte {
		t.Helper()
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	screens := []string{"80x24-no-color", "140x40-no-color"}
	golden := func(root, screen string) string {
		return filepath.Join(root, "huginn", "internal", "tui", "testdata", "tmux", screen, "palette.txt")
	}
	// tree lays out the run's goldens, with edit applied to the 140x40 one.
	tree := func(edit func([]byte) []byte) string {
		t.Helper()
		root := t.TempDir()
		for _, s := range screens {
			b := read(filepath.Join(fixture, "want", s+".txt"))
			if s == "140x40-no-color" {
				b = edit(b)
			}
			if err := os.MkdirAll(filepath.Dir(golden(root, s)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden(root, s), b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return root
	}
	run := func(root string, args ...string) (int, string) {
		t.Helper()
		cmd := exec.Command("bash", append([]string{script, "--log", filepath.Join(fixture, "failed.log")}, args...)...)
		cmd.Dir = root
		out, _ := cmd.CombinedOutput()
		return cmd.ProcessState.ExitCode(), string(out)
	}

	for _, job := range [][]string{nil, {"test (ubuntu-latest)"}} {
		root := tree(func(b []byte) []byte { return b })
		if code, out := run(root, job...); code != 0 {
			t.Fatalf("job %q: exit %d:\n%s", job, code, out)
		}
		for _, s := range screens {
			if got, want := read(golden(root, s)), read(filepath.Join(fixture, "got", s+".txt")); !bytes.Equal(got, want) {
				t.Fatalf("job %q: %s is not the screen CI captured:\n got %q\nwant %q", job, s, got, want)
			}
		}
	}

	// One changed byte in a golden: the log is not about this tree. Nothing
	// is written, not even the golden that does match.
	root := tree(func(b []byte) []byte { return bytes.Replace(b, []byte("GLOBAL"), []byte("GLOBAl"), 1) })
	code, out := run(root)
	if code == 0 || !strings.Contains(out, "wrote nothing") || !strings.Contains(out, "\nfix: ") {
		t.Fatalf("a tampered golden passed (exit %d):\n%s", code, out)
	}
	if got, want := read(golden(root, "80x24-no-color")), read(filepath.Join(fixture, "want", "80x24-no-color.txt")); !bytes.Equal(got, want) {
		t.Fatalf("80x24 was rewritten although the run failed its check")
	}

	// A log without drift blocks says so, with a fix line.
	empty := filepath.Join(t.TempDir(), "empty.log")
	if err := os.WriteFile(empty, []byte("test (ubuntu-latest)\tTest\t2026-11-18T21:45:35.7605694Z FAIL\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", script, "--log", empty)
	cmd.Dir = tree(func(b []byte) []byte { return b })
	if b, _ := cmd.CombinedOutput(); cmd.ProcessState.ExitCode() == 0 || !strings.Contains(string(b), "no 'golden drift' block") || !strings.Contains(string(b), "\nfix: ") {
		t.Fatalf("an empty log passed (exit %d):\n%s", cmd.ProcessState.ExitCode(), b)
	}
}

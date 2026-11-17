package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestCacheWriteRacesReads (557s7n): setup's remembered checks are
// replaced while other hug commands read them. Windows CI 44172854884
// lost one ("remembered files check: []" with a valid key and a clean
// board): writeAtomic renamed a fixed path.tmp with os.Rename and ignored
// the error. Two writers and four readers on one file, as kbc99j's probe:
// no write and no read fails, and every read is one whole version.
func TestCacheWriteRacesReads(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "huginn", "setup-files-x.json")
	a, b := bytes.Repeat([]byte("a"), 4096), bytes.Repeat([]byte("b"), 8192)
	if e := writeAtomic(path, a); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(2 * time.Second)
	var mu sync.Mutex
	var readErrs, writeErrs []error
	torn, reads, writes := 0, 0, 0
	var wg sync.WaitGroup
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for time.Now().Before(deadline) {
				got, e := readCache(path)
				mu.Lock()
				reads++
				if e != nil {
					readErrs = append(readErrs, e)
				} else if !bytes.Equal(got, a) && !bytes.Equal(got, b) {
					torn++
				}
				mu.Unlock()
			}
		}()
	}
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; time.Now().Before(deadline); i++ {
				data := a
				if i%2 == 1 {
					data = b
				}
				e := writeAtomic(path, data)
				mu.Lock()
				writes++
				if e != nil {
					writeErrs = append(writeErrs, e)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	t.Logf("%d reads, %d writes", reads, writes)
	first := func(es []error) error {
		if len(es) == 0 {
			return nil
		}
		return es[0]
	}
	if len(readErrs) > 0 || len(writeErrs) > 0 || torn > 0 {
		t.Fatalf("%d of %d reads failed (first: %v), %d of %d writes failed (first: %v), %d torn reads",
			len(readErrs), reads, first(readErrs), len(writeErrs), writes, first(writeErrs), torn)
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*"))
	if len(left) != 1 {
		t.Fatalf("temp files left behind: %v", left)
	}
}

// A remembered check that cannot be written leaves setup's answer as it
// is, and says so on stderr once, with a fix (557s7n). A folder where the
// file goes makes the rename fail on every OS.
func TestSetupSaysWhenItCannotRemember(t *testing.T) {
	t.Parallel()
	// Both of setup's remembered checks (backlog, task files) fail in one
	// run: still one line.
	a, _ := backlogRepo(t)
	id := execute(t, a, "new", "Has evidence", "--goal", "G", "--purpose", "P", "--principle", "Q", "--accept", "C")["task"].(Reply)["id"].(string)
	ev := filepath.Join(a.Cwd, "evidence", id)
	if e := os.MkdirAll(ev, 0o755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(ev, "run.log"), []byte("ok\n"), 0o644); e != nil {
		t.Fatal(e)
	}
	blocker := filesCheckFile(filepath.Join(a.Cwd, ".git"), "evidence")
	for _, p := range []string{blocker, backlogCheckFile(filepath.Join(a.Cwd, ".git"))} {
		if e := os.MkdirAll(filepath.Join(p, "in-the-way"), 0o755); e != nil {
			t.Fatal(e)
		}
	}
	for round := 1; round <= 2; round++ {
		s, errs := setupStep(t, a, "import task files")
		if !strings.Contains(s.Detail, "1 files to copy, 0 already in place") {
			t.Fatalf("round %d: setup's answer changed: %s", round, s.Detail)
		}
		if !strings.Contains(errs, "comparing .backlog/") {
			t.Fatalf("round %d: the backlog check did not run:\n%s", round, errs)
		}
		if n := strings.Count(errs, "could not remember"); n != 1 {
			t.Fatalf("round %d: want one 'could not remember' line on stderr, got %d:\n%s", round, n, errs)
		}
		if !strings.Contains(errs, "fix: ") {
			t.Fatalf("round %d: no fix: line:\n%s", round, errs)
		}
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(blocker), "*.tmp"))
	if len(left) != 0 {
		t.Fatalf("temp files left behind: %v", left)
	}
	// Each fix line clears its own path: after them both checks are
	// remembered.
	if runtime.GOOS == "windows" {
		return
	}
	for i := 0; ; i++ {
		_, errs := setupStep(t, a, "import task files")
		_, fix, found := strings.Cut(errs, "fix: ")
		if !found {
			break
		}
		if i == 2 {
			t.Fatalf("still failing after 2 fix lines:\n%s", errs)
		}
		fix, _, _ = strings.Cut(fix, " && hug task setup")
		if !strings.HasPrefix(fix, "rm -rf '") {
			t.Fatalf("fix: %q", fix)
		}
		if out, e := exec.Command("sh", "-c", fix).CombinedOutput(); e != nil {
			t.Fatalf("%s: %v %s", fix, e, out)
		}
	}
	for _, p := range []string{blocker, backlogCheckFile(filepath.Join(a.Cwd, ".git"))} {
		if fi, e := os.Stat(p); e != nil || !fi.Mode().IsRegular() {
			t.Fatalf("after the fix lines %s is still not remembered: %v %v", p, fi, e)
		}
	}
}

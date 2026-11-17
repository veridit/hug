package task

import (
	"bytes"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/veridit/hug/internal/sharefile"
)

// TestAtomicWriteRacesReads: kbc99j: a task file read while another writer
// renames a new version over it must neither fail nor fail the rename. On
// Windows (CI run 44086864807, before the fix) 6 of 11646 os.ReadFile calls
// failed with "The process cannot access the file because it is being used
// by another process" and 165 of 169 renames with "Access is denied": the
// failure of parallel_test.go:34 (run 44065882953). Both sides are counted
// so the failure says which one lost.
func TestAtomicWriteRacesReads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "h.race.md")
	a, b := bytes.Repeat([]byte("a"), 4096), bytes.Repeat([]byte("b"), 8192)
	if err := AtomicWrite(path, a); err != nil {
		t.Fatal(err)
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
				got, err := sharefile.ReadFile(path)
				mu.Lock()
				reads++
				if err != nil {
					readErrs = append(readErrs, err)
				} else if !bytes.Equal(got, a) && !bytes.Equal(got, b) {
					torn++
				}
				mu.Unlock()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; time.Now().Before(deadline); i++ {
			data := a
			if i%2 == 1 {
				data = b
			}
			err := AtomicWrite(path, data)
			mu.Lock()
			writes++
			if err != nil {
				writeErrs = append(writeErrs, err)
			}
			mu.Unlock()
		}
	}()
	wg.Wait()
	t.Logf("%d reads, %d writes", reads, writes)
	if len(readErrs) > 0 || len(writeErrs) > 0 || torn > 0 {
		first := func(es []error) error {
			if len(es) == 0 {
				return nil
			}
			return es[0]
		}
		t.Fatalf("%d of %d reads failed (first: %v), %d of %d writes failed (first: %v), %d torn reads",
			len(readErrs), reads, first(readErrs), len(writeErrs), writes, first(writeErrs), torn)
	}
}

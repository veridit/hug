package task

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// kbc99j: a task file read while another process renames a new version
// over it must neither fail nor fail the rename. On Windows (CI run
// 44065882953) a parallel note failed with "open ...: The process cannot
// access the file because it is being used by another process": a read
// outside the board turn met a writer's rename. Both sides are counted so
// the failure says which one lost.
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
				got, err := ReadTaskFile(path)
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

// ReadTaskFile is the read under test; until the fix it is os.ReadFile.
var ReadTaskFile = os.ReadFile

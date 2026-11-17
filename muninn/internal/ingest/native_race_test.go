package ingest

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNativeSaveRacesLoads (zd3x24): readers load the native store while a
// writer saves it, as recall hooks and capture do in separate processes. On
// Windows a plain read held the file without FILE_SHARE_DELETE, so the
// save's rename over it failed ("Access is denied") and capture lost its
// batch (capture_batch_test.go:200, main CI 44092559068). Every load and
// every save must succeed.
func TestNativeSaveRacesLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memories.json")
	now := time.Now()
	mem := func(i int) NativeMemory {
		return NativeMemory{ID: fmt.Sprintf("race_%d", i), Content: fmt.Sprintf("Synthetic race memory %d holds.", i), Category: "fact", Active: true, Strength: 1, Created: now, Updated: now}
	}
	if err := SaveNative(path, []NativeMemory{mem(0)}); err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	var reads, readErrs atomic.Int64
	var firstRead atomic.Value
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				reads.Add(1)
				if _, err := LoadNative(path); err != nil {
					readErrs.Add(1)
					firstRead.CompareAndSwap(nil, err.Error())
				}
			}
		}()
	}
	writes, writeErrs, firstWrite := 0, 0, ""
	mems := []NativeMemory{mem(0)}
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		writes++
		mems = append(mems, mem(writes))
		if err := SaveNative(path, append([]NativeMemory(nil), mems...)); err != nil {
			writeErrs++
			if firstWrite == "" {
				firstWrite = err.Error()
			}
		}
	}
	close(stop)
	wg.Wait()
	t.Logf("%d reads, %d writes", reads.Load(), writes)
	if readErrs.Load() > 0 || writeErrs > 0 {
		t.Fatalf("%d of %d loads failed (first: %v), %d of %d saves failed (first: %s)", readErrs.Load(), reads.Load(), firstRead.Load(), writeErrs, writes, firstWrite)
	}
}

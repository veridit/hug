package run

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
)

// 4vd1mz check 1: hug mem forget ID does no full index pass when the index
// is current; it updates what the forget changed (Indexer.ApplyNative). On
// main it did 2: one before the forget and one after the write. A batch
// forget is one bounded update too. When the store changed since the last
// pass (another process wrote without indexing), forget runs exactly one
// pass first, so it still resolves and forgets what is on disk now.
func TestForgetRunsNoIndexPass(t *testing.T) {
	home := memHome(t)
	ids := rememberN(t, 6)
	if _, _, code := runCLI(t, "", "index"); code != 0 {
		t.Fatal("index failed")
	}
	t.Cleanup(func() { openApp = app.Open })
	// count runs one command and returns how many full passes its app ran.
	count := func(args ...string) int {
		t.Helper()
		var last *app.App
		openApp = func(ctx context.Context) (*app.App, error) {
			a, err := app.Open(ctx)
			last = a
			return a, err
		}
		out, errOut, code := runCLI(t, "", args...)
		if code != 0 || last == nil {
			t.Fatalf("%v: %s%s", args, out, errOut)
		}
		return last.Indexer.Passes
	}
	if n := count("forget", ids[0]); n != 0 {
		t.Errorf("hug mem forget ID ran %d index passes, want 0", n)
	}
	if n := count("forget", ids[1], ids[2]); n != 0 {
		t.Errorf("hug mem forget A B ran %d index passes, want 0", n)
	}
	// Another process writes the store and does not index: one pass first.
	if _, _, err := ingest.Remember(filepath.Join(home, "data", "memories.json"), "4vd1mz written behind the index's back", "fact", "global", nil); err != nil {
		t.Fatal(err)
	}
	if n := count("forget", ids[3]); n != 1 {
		t.Errorf("forget after an unindexed write ran %d passes, want 1", n)
	}
	// What forget changed is what a pass would find: a pass now has no work.
	out, errOut, code := runCLI(t, "", "index")
	if code != 0 || !strings.Contains(errOut+out, "changed=0") || !strings.Contains(errOut+out, "scanned=0") {
		t.Fatalf("a pass after targeted forgets still found work: %s%s", out, errOut)
	}
	out, _, _ = runCLI(t, "", "list", "--json")
	for _, id := range ids[:4] {
		if strings.Contains(out, `"id":"`+id+`"`) {
			t.Errorf("forgotten %s still listed", id)
		}
	}
}

// 4vd1mz: one forget reads the store file once (to change it) when no other
// writer touched it after the save: the readback (tn8bmn) sees the file at
// the stat its own save left, under the lock, and trusts what it wrote. On
// main it read it twice. TestForgetUndoneByAnotherWriterFails keeps the
// other side: a writer in between makes the readback load and fail.
func TestForgetReadsStoreOnce(t *testing.T) {
	memHome(t)
	ids := rememberN(t, 3)
	if _, _, code := runCLI(t, "", "index"); code != 0 {
		t.Fatal("index failed")
	}
	before := ingest.Loads.Load()
	if out, errOut, code := runCLI(t, "", "forget", ids[0]); code != 0 {
		t.Fatalf("forget: %s%s", out, errOut)
	}
	if n := ingest.Loads.Load() - before; n != 1 {
		t.Errorf("hug mem forget ID loaded the store %d times, want 1", n)
	}
}

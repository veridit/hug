package run

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/ingest"
)

// t0qap9 check 6: hug mem remember does no full index pass when the index
// is current; it indexes what its write changed (Indexer.ApplyWritten). On
// main every remember ran a full pass after its write (binding.go:778), so
// on a store with pending work a remember did that whole pass under its own
// write turn. A reinforce (the same text again) is targeted too. When
// another process wrote the store without indexing, remember runs one pass,
// so what is on disk now is indexed.
func TestRememberRunsNoIndexPass(t *testing.T) {
	home := memHome(t)
	rememberN(t, 4)
	if _, _, code := runCLI(t, "", "index"); code != 0 {
		t.Fatal("index failed")
	}
	t.Cleanup(func() { openApp = app.Open })
	count := func(args ...string) (int, string) {
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
		return last.Indexer.Passes, out
	}
	n, out := count("remember", "t0qap9 a remember indexes only its own write")
	if n != 0 {
		t.Errorf("hug mem remember ran %d index passes, want 0", n)
	}
	f := strings.Fields(out)
	if len(f) < 2 || f[0] != "remembered" {
		t.Fatalf("remember: %q", out)
	}
	if n, out := count("remember", "t0qap9 a remember indexes only its own write"); n != 0 || !strings.HasPrefix(out, "reinforced "+f[1]) {
		t.Errorf("a reinforcing remember ran %d index passes (want 0): %q", n, out)
	}
	if n, _ := count("remember", "t0qap9 a remember with a trigger", "--when", "checking how remember indexes"); n != 0 {
		t.Errorf("a remember with a trigger ran %d index passes, want 0", n)
	}
	// What remember indexed is what a pass would find: a pass has no work.
	out, errOut, code := runCLI(t, "", "index")
	if code != 0 || !strings.Contains(errOut+out, "changed=0") || !strings.Contains(errOut+out, "scanned=0") {
		t.Fatalf("a pass after targeted remembers still found work: %s%s", out, errOut)
	}
	// The remembered memory is found by search, as a pass would leave it.
	if out, _, code := runCLI(t, "", "search", "indexes only its own write"); code != 0 || !strings.Contains(out, f[1]) {
		t.Fatalf("search does not find %s: %s", f[1], out)
	}
	// Another process writes the store and does not index: one pass.
	if _, _, err := ingest.Remember(filepath.Join(home, "data", "memories.json"), "t0qap9 written behind the index's back", "fact", "global", nil); err != nil {
		t.Fatal(err)
	}
	if n, _ := count("remember", "t0qap9 remember after an unindexed write"); n != 1 {
		t.Errorf("remember after an unindexed write ran %d passes, want 1", n)
	}
	out, errOut, code = runCLI(t, "", "index")
	if code != 0 || !strings.Contains(errOut+out, "scanned=0") {
		t.Fatalf("a pass after the fallback still found work: %s%s", out, errOut)
	}
}

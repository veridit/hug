// Package timing gates the tests that hold a wall-clock budget (khq1fq).
//
// A wall-clock budget fails a test when something took longer than a
// fixed time. CI judges it, where a miss is retried before the change
// lands. release.yml rebuilds a SHA CI already passed: it sets
// HUG_SKIP_TIMING=1, and every budget test calls Budget first, so a
// loaded release runner cannot fail a release on speed (v0.7.113, run
// 45627554531: TestBigTaskKeys p50 53.8 ms over 50 ms after CI
// 45625116019 passed). internal/repocheck keeps every budget test calling
// it and release.yml setting it.
package timing

import (
	"os"
	"testing"
)

// Skip is the variable that skips the wall-clock budget tests.
const Skip = "HUG_SKIP_TIMING"

// Skipped reports whether HUG_SKIP_TIMING=1 is set.
func Skipped() bool { return os.Getenv(Skip) == "1" }

// Budget skips t when HUG_SKIP_TIMING=1: t holds a wall-clock budget.
// Otherwise it logs that the budget is judged here, so a -v log (ci.yml)
// names every budget test that ran.
func Budget(t testing.TB) {
	t.Helper()
	if Skipped() {
		t.Skipf("wall-clock budget %s: skipped under %s=1 (release.yml); CI judges it", t.Name(), Skip)
	}
	t.Logf("wall-clock budget %s: judged here (%s is not 1)", t.Name(), Skip)
}

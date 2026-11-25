package timing

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// khq1fq: Budget skips under HUG_SKIP_TIMING=1 and runs the test
// otherwise (unset, empty, 0). Each case runs Budget in a child test
// process, so a skip is seen from outside, as go test reports it.
func TestBudgetSkipsOnlyUnderSkipTiming(t *testing.T) {
	if os.Getenv("HUG_CHILD_ARGS") == "budget" {
		Budget(t)
		t.Log("BUDGET RAN")
		return
	}
	for _, c := range []struct {
		value string
		set   bool
		skip  bool
	}{
		{"1", true, true},
		{"", false, false},
		{"", true, false},
		{"0", true, false},
	} {
		cmd := exec.Command(os.Args[0], "-test.run=^TestBudgetSkipsOnlyUnderSkipTiming$", "-test.v", "-test.count=1")
		env := []string{"HUG_CHILD_ARGS=budget"}
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(kv, Skip+"=") && !strings.HasPrefix(kv, "HUG_CHILD_ARGS=") {
				env = append(env, kv)
			}
		}
		if c.set {
			env = append(env, Skip+"="+c.value)
		}
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s=%q (set %v): child failed: %v\n%s", Skip, c.value, c.set, err, out)
		}
		skipped := strings.Contains(string(out), "--- SKIP: TestBudgetSkipsOnlyUnderSkipTiming")
		ran := strings.Contains(string(out), "BUDGET RAN")
		if skipped != c.skip || ran == c.skip {
			t.Errorf("%s=%q (set %v): skipped %v, ran %v, want skipped %v\n%s", Skip, c.value, c.set, skipped, ran, c.skip, out)
		}
		why := "wall-clock budget TestBudgetSkipsOnlyUnderSkipTiming: skipped under HUG_SKIP_TIMING=1"
		if !c.skip {
			why = "wall-clock budget TestBudgetSkipsOnlyUnderSkipTiming: judged here"
		}
		if !strings.Contains(string(out), why) {
			t.Errorf("%s=%q (set %v): the log lacks %q:\n%s", Skip, c.value, c.set, why, out)
		}
	}
}

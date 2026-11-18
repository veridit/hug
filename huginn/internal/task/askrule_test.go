package task

import "testing"

// 41hhgn check 13: which waits hug upgrade reads as a question to a
// person, on every waiting_for wait the boards held at v0.7.52: hug's 7
// (commit 8a18427b on hug/tasks) and Statbus's 5. A '?', a check named or
// a request verb first moves it to asked; anything else stays a wait,
// listed with the ask that would move it.
func TestReadsAsQuestion(t *testing.T) {
	for _, c := range []struct {
		what string
		want bool
	}{
		// hug
		{"H3 failed Part 2 (10 wrong vs bar 6); parked per crab ruling, no fourth arm", false},
		{"approve the design (attached: bxqtkt.md), check 1, and answer Q1-Q6 at its end", true},
		{"a week of use: has he had to remind an agent to update a ticket? look again 2026-11-24", true},
		{"check 4: do the status colours read as one subtle scheme? The capture shows the overview", true},
		{"check 3: find a Statbus task by an old id you remember (/ STATBUS-402 in hug tui on ~/ssb/statbus)", true},
		{"check 1: open an image in iTerm2 (Space on a .png in FILES), and check 4: answer zhvkmj's colour question", true},
		{"approve Seid's commit-msg switch to hug's Model: <id>/<effort> gate (3 decisions in the file)", true},
		// Statbus: a release and two owner tasks stay waits (no '?', no
		// check, no request first); each is listed with its ask.
		{"the hug release that stops linting done tasks (installed now: v0.7.50)", false},
		{"owner looks at the new-record prototype on LU 31/28 contact", false},
		{"owner confirms time frames on Norway after the release (rc.02 shipped)", false},
		{"check #4: owner removes a test record on Norway after the release carrying this", true},
		{"Albania's StatBus version: the owner asks them (again 2026-11-18); v2026.11.0 or later lets check 3 be ticked", true},
		// A check the task does not have names nothing.
		{"see check 12 of the epic", false},
	} {
		if got := readsAsQuestion(c.what, 5); got != c.want {
			t.Errorf("readsAsQuestion(%q) = %v, want %v", c.what, got, c.want)
		}
	}
}

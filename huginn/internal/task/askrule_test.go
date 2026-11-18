package task

import "testing"

// 41hhgn check 13: which waits hug upgrade reads as a question to a
// person, on every waiting_for wait the boards held at v0.7.52: hug's 7
// (commit 8a18427b on hug/tasks) and Statbus's 5. A '?', a check named or
// a request verb in the first clause moves it to asked; anything else
// stays a wait, listed with the ask that would move it.
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
		// Statbus: the owner is asked to look or confirm (crab 02:18: a
		// request verb anywhere in the first clause, any person form);
		// a release is a thing, it stays a wait.
		{"the hug release that stops linting done tasks (installed now: v0.7.50)", false},
		{"owner looks at the new-record prototype on LU 31/28 contact", true},
		{"owner confirms time frames on Norway after the release (rc.02 shipped)", true},
		{"check #4: owner removes a test record on Norway after the release carrying this", true},
		{"Albania's StatBus version: the owner asks them (again 2026-11-18); v2026.11.0 or later lets check 3 be ticked", true},
		// A check the task does not have names nothing.
		{"see check 12 of the epic", false},
		// Inflected forms: base, -s and -ies ask; -ed and -ing report.
		{"jhf reviews the TUI capture", true},
		{"Jørgen approves the layout", true},
		{"crab decides between the two arms", true},
		{"owner verifies the import on Norway", true},
		{"approved by crab 01:20, waiting for the release", false},
		{"parked: crab decided no fourth arm", false},
		{"CI run 9 on main", false},
		// Noun senses stay out of the list: a park under a rule is a park.
		{"parked under crab's rule for H3 arms", false},
		{"CI test 9 on main", false},
		// Only the first clause: a verb after a comma is context.
		{"parked until v0.7.53, then jhf reviews it", false},
	} {
		if got := readsAsQuestion(c.what, 5); got != c.want {
			t.Errorf("readsAsQuestion(%q) = %v, want %v", c.what, got, c.want)
		}
	}
}

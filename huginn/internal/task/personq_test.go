package task

import "testing"

// agxqax: which waits are questions to a person, and who for. Every case
// is a real waiting_on from hug's, Statbus's or sql_saga's board history,
// or the shape the goal names.
func TestAddressesPerson(t *testing.T) {
	people := []Person{{Short: "jhf", Name: "Jørgen H. Fjeld", Alias: []string{"king"}}, {Short: "erik", Name: "Erik Søberg"}, {Short: "hhz", Name: "Hedda"}}
	for _, c := range []struct {
		on, want string
		check    int
	}{
		{"owner: check 4 on Norway rc.02 (remove a test record through the form)", "jhf", 4},
		{"owner: check 6 (plain paging, frame-grouped, or both)", "jhf", 6},
		{"the owner's decision on hug upgrade --apply (hug v0.7.53 is installed)", "jhf", 0},
		{"owner confirms time frames on Norway after the release (rc.02 shipped)", "jhf", 0},
		{"check #4: owner removes a test record on Norway after the release carrying this", "jhf", 4},
		{"jhf: check 9: look at the stage rows (step 4 waits on the exclusive main window)", "jhf", 9},
		{"Jørgen's answer on footer-mock.md (asked 05:19), then build", "jhf", 0},
		{"Jorgen: approve the colours", "jhf", 0},
		{"king: approve the colours", "jhf", 0},
		{"Erik Søberg: the import mapping", "erik", 0},
		{"Hedda looks at the new form", "hhz", 0},
		{"waiting for the owner to merge PR 12", "jhf", 0},
		// ebben8: a person in [people] as the subject, whatever the verb
		// (hug's open checks of 2026-11-18 that the verb list missed).
		{"Jørgen attaches a screenshot to a task from hug (or the CLI) and the detail shows it inline in iTerm2 and kitty, ANSI captures with colours anywhere, one line elsewhere (golden for the fallback)", "jhf", 0},
		{"Jørgen writes a real request on hug's board after release and an agent resolves it into ticket changes he agrees with", "jhf", 0},
		{"Jørgen links to a task by its path from outside hug and the link survives a reorder", "jhf", 0},
		{"Jørgen sees 47sxn2 come back into his ASKED on 2026-11-24 without anyone touching it", "jhf", 0},
		{"Jørgen writes a request in hug tui and the coordinator reports receiving it, unprompted.", "jhf", 0},
		{"Jørgen reloads after an upgrade, opens a task and sees no leftover line.", "jhf", 0},
		{"Jørgen reads 911j70's comparison sheets from hug tui with zoom or Preview and answers check 1", "jhf", 0},
		{"check 1: Jørgen sees the Seid overview preview; check 3: rows show tags (not started, mizaru moved to q0a8rx by crab 19:44Z)", "jhf", 1},
		{"Jørgen H. Fjeld attaches a screenshot", "jhf", 0},
		{"Erik Søberg imports the mapping", "erik", 0},
		{"Jørgen, after the release, attaches a screenshot", "jhf", 0},
		{"Jørgen and crab go through it", "jhf", 0},
		{"the King signs off on it", "jhf", 0},
		// Not questions to a person: things, agents, people in passing.
		{"CI run 7", "", 0},
		{"crab: landing queue after qpp30y", "", 0},
		{"crab review; check 3 with puppy's grid, check 4 Jørgen", "", 0},
		{"tje4sx first (Jørgen's ruling); resumes after it lands", "", 0},
		{"the release carrying 0c3d3924 and 'hug upgrade --apply' on Jørgen's machine, then a day of jcode use for check 1", "", 0},
		{"check 3: dream_last set on Jørgen's machine within a day of the release holding 1151adad", "", 0},
		{"Albania's StatBus version: the owner asks them (again 2026-11-18)", "", 0},
		{"check 5: order the Statbus overview by priority, then by owner, with Tab and o (f003cfc), and say it works", "", 0},
		{"the date 2026-11-24, then ask jhf", "", 0},
		{"PR approved by jhf, CI run 9", "", 0},
		{"the user table migration", "", 0},
		{"bob: approve it", "", 0},
		// ebben8: a person named, but not as who acts.
		{"Jørgen 19:35Z: tags sit right after the task ID (pie, id, tags, type, priority, stage, text), dim, at most 22 columns", "", 0},
		{"Jørgen's machine shows the stale-server line", "", 0},
		{"Hedda's form opens", "", 0},
		{"the owner table has a column", "", 0},
		{"JHF.md lists the tasks", "", 0},
		{"jhf/tasks is a branch", "", 0},
		{"Jørgen", "", 0},
		{"a mock reviewed by Jørgen", "", 0},
	} {
		w, ok := AddressesPerson(people, c.on, 9)
		if ok != (c.want != "") || w.For != c.want || w.Check != c.check {
			t.Errorf("%q: got %+v %v, want for %q check %d", c.on, w, ok, c.want, c.check)
		}
	}
	// No [people]: nobody to ask, nothing refused.
	if _, ok := AddressesPerson(nil, "owner: check 4", 9); ok {
		t.Error("a board without [people] refused a wait")
	}
}

// agxqax check 2: a person is found by short name, alias, long name or
// first name, folded; a first name two people share finds neither.
func TestPersonMatch(t *testing.T) {
	people := []Person{{Short: "jhf", Name: "Jørgen H. Fjeld"}, {Short: "erik", Name: "Erik Søberg"}, {Short: "eh", Name: "Erik Hansen"}, {Short: "hhz", Name: "Hedda", Alias: []string{"hedda-ssb"}}}
	for name, want := range map[string]string{"jhf": "jhf", "JHF": "jhf", "@jhf": "jhf", "jorgen": "jhf", "Jørgen": "jhf", "Jørgen H. Fjeld": "jhf", "jorgen h fjeld": "jhf", "hedda-ssb": "hhz", "hedda": "hhz", "Erik Søberg": "erik", "erik": "erik", "erik hansen": "eh", "bob": "", "": ""} {
		p, ok := PersonMatch(people, name)
		if ok != (want != "") || p.Short != want {
			t.Errorf("%q: got %q %v, want %q", name, p.Short, ok, want)
		}
	}
}

package task

import (
	"os"
	"path/filepath"
	"testing"
)

// atex3d (z2vmte check 5, Jørgen 2026-11-20 13:05): a wait for a date
// leads with the date, "waiting until 2026-11-24 · on WHY", or for a held
// question "· then ask NAME", with no second "until" and no "on the date".
// A task written by an older hug (waiting_on "the date ..., then ask ...")
// reads the new way too.
func TestWaitingLineLeadsWithTheDate(t *testing.T) {
	setClock(t, 2026, 11, 18)
	s := &Store{Root: t.TempDir()}
	a := doingTask(t, s, "aaaaaa", "kestrel")
	if _, e := s.WaitUntil(a, "a week of use", "2026-11-24", "kestrel"); e != nil {
		t.Fatal(e)
	}
	b := doingTask(t, s, "bbbbbb", "crab")
	if _, e := s.AskAfter(b, Question{What: "has he had to remind an agent?", For: "jhf", Check: 1}, "2026-11-24", "crab"); e != nil {
		t.Fatal(e)
	}
	// As v0.7.55 wrote 47sxn2 on hug's board (2026-11-18 05:08).
	old := "---\nid: \"cccccc\"\ntitle: \"Old\"\nstate: \"doing\"\nstage: \"waiting\"\nassignee: [\"crab\"]\ndoing_since: \"2026-11-17T18:10\"\nwaiting_since: \"2026-11-18T05:08\"\nwaiting_until: \"2026-11-24\"\nwaiting_ask_what: \"a week of use: has he had to remind an agent to update a ticket?\"\nwaiting_ask_for: \"jhf\"\nwaiting_asked_by: \"crab\"\nwaiting_ask_check: \"1\"\nwaiting_on: \"the date 2026-11-24, then ask jhf\"\ntype: \"feature\"\npriority: \"medium\"\nlabels: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Acceptance\n- [ ] a\n"
	if e := os.WriteFile(filepath.Join(s.Root, "cccccc.t.md"), []byte(old), 0o644); e != nil {
		t.Fatal(e)
	}
	d := doingTask(t, s, "dddddd", "kestrel")
	if _, e := s.Wait(d, "CI run 7", "kestrel"); e != nil {
		t.Fatal(e)
	}
	// The held question decides, whatever waiting_on says; with its ask
	// fields gone, the stored "the date ..., " is still not shown.
	front := func(id, fields string) {
		body := "---\nid: \"" + id + "\"\ntitle: \"T\"\nstate: \"doing\"\nstage: \"waiting\"\nassignee: [\"crab\"]\nwaiting_since: \"2026-11-18T05:08\"\nwaiting_until: \"2026-11-24\"\n" + fields + "type: \"feature\"\npriority: \"medium\"\nlabels: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n\n## Acceptance\n- [ ] a\n"
		if e := os.WriteFile(filepath.Join(s.Root, id+".t.md"), []byte(body), 0o644); e != nil {
			t.Fatal(e)
		}
	}
	front("eeeeee", "waiting_ask_what: \"q?\"\nwaiting_ask_for: \"jhf\"\nwaiting_on: \"a week of use\"\n")
	front("ffffff", "waiting_on: \"the date 2026-11-24, then ask jhf\"\n")
	front("gggggg", "")
	for id, want := range map[string]string{
		a:        "aaaaaa waiting until 2026-11-24 · on a week of use",
		b:        "bbbbbb waiting until 2026-11-24 · then ask jhf",
		"cccccc": "cccccc waiting until 2026-11-24 · then ask jhf",
		d:        "dddddd waiting on CI run 7", // no date: as before
		"eeeeee": "eeeeee waiting until 2026-11-24 · then ask jhf",
		"ffffff": "ffffff waiting until 2026-11-24 · then ask jhf",
		"gggggg": "gggggg waiting until 2026-11-24",
	} {
		tk, _ := load(t, s, id)
		if got := WaitingLine(tk); got != want {
			t.Errorf("WaitingLine(%s) = %q, want %q", id, got, want)
		}
	}
}

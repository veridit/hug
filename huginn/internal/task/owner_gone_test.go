package task

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// 9ahvgt (retriever 2026-11-18 12:34Z): lint said "no findings with code
// stale-owner" while 26+ doing tasks belonged to gone sessions. Three
// shapes from ~/.jcode/sessions that day:
//   - nautilus: reaped 07:46 ("Reaped idle spawned swarm worker ... idle >
//     1800s"), its last journal line says "status":"Closed"; only 7 h ago,
//     so an mtime-only 24 h rule calls it live.
//   - the newest write is the .journal.jsonl, not the .json.
//   - kestrel: taken --as kestrel by session calf; no session file named
//     kestrel anywhere, so the name was never judged.
//
// A live worker (an Active session written minutes ago) is never reported.
func TestOwnerFindingsGoneSessions(t *testing.T) {
	dir, other := t.TempDir(), t.TempDir()
	repo := "/Users/jhf/Projects/hug"
	now := time.Date(2026, 11, 18, 14, 0, 0, 0, time.UTC)
	write := func(name, status string, at time.Time, title ...string) {
		t.Helper()
		// The shape jcode writes: top-level short_name then status (.json),
		// or one {"meta":{...}} line per save (.journal.jsonl).
		dir, wd := dir, repo
		if d, n, ok := strings.Cut(name, "/"); ok { // another jcode instance
			dir, wd, name = filepath.Join(other, d), "/elsewhere/"+d, n
		}
		meta := `"working_dir":"` + wd + `","short_name":"` + strings.Split(name, "_")[1] + `","status":"`
		body := `{"id":"x","title":"` + strings.Join(title, "") + `","messages":[{"text":"\"short_name\":\"q\",\"status\":\"Active\""}],` + meta + status + `","last_pid":1}`
		if strings.HasSuffix(name, ".journal.jsonl") {
			body = `{"meta":{` + meta + `Active"},"append_messages":[]}` + "\n" + `{"meta":{` + meta + status + `"}}` + "\n"
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	// nautilus: json Active at 07:07, journal Closed at 07:46 (reaped).
	write("session_nautilus_1794905151412_33a1.json", "Active", now.Add(-7*time.Hour))
	write("session_nautilus_1794905151412_33a1.journal.jsonl", "Closed", now.Add(-6*time.Hour-14*time.Minute))
	// hatchling: closed json only.
	write("session_hatchling_1794905151586_536f.json", "Closed", now.Add(-7*time.Hour-13*time.Minute))
	// calf, told "You are kestrel" in its first message, took as kestrel:
	// no file is named kestrel; the title is how hug finds it.
	write("session_calf_1794909717965_ba2b.json", "Closed", now.Add(-5*time.Hour), "You are kestrel, a worker on hug (~/Projects/hug); read AGENTS.m\u2026")
	// macaque: a live worker, written a minute ago; an older closed
	// macaque session of another project does not make it gone.
	write("session_macaque_1795000000000_aaaa.journal.jsonl", "Active", now.Add(-time.Minute))
	write("session_macaque_1794000000000_bbbb.json", "Closed", now.Add(-3*time.Hour))
	// Session names repeat across jcode instances (2026-11-18: calf and
	// wyvern live in Statbus's at 14:23 and 14:33): a session in another
	// repository says nothing about this board's owners.
	write("ssb/session_calf_1795008207959_2309.json", "Active", now.Add(-time.Minute), "You are kestrel, a Statbus worker")
	write("ssb/session_wyvern_1795008787421_c706.journal.jsonl", "Active", now.Add(-time.Minute))
	write("session_wyvern_1794905939974_6ffa.journal.jsonl", "Closed", now.Add(-3*time.Hour-47*time.Minute))
	// puppy: closed 20 minutes ago, after reporting; not yet gone.
	write("session_puppy_1795000793237_946e.journal.jsonl", "Closed", now.Add(-20*time.Minute))
	ts := []Task{
		{ID: "zhvkmj", State: "doing", Assignee: []string{"nautilus"}, Path: "z/zhvkmj.md"},
		{ID: "tb0dra", State: "doing", Assignee: []string{"hatchling"}, Path: "t/tb0dra.md"},
		{ID: "bxqtkt", State: "doing", Assignee: []string{"kestrel"}, Path: "b/bxqtkt.md", Model: "claude-opus-5-5"},
		{ID: "sortxx", State: "doing", Assignee: []string{"macaque"}, Path: "s/sortxx.md"},
		{ID: "911j70", State: "doing", Assignee: []string{"puppy"}, Path: "9/911j70.md"},
		{ID: "jr55s8", State: "doing", Assignee: []string{"wyvern"}, Path: "j/jr55s8.md"},
		{ID: "t7wetj", State: "doing", Assignee: []string{"jhf"}, Path: "t/t7wetj.md", Model: "human"},                       // a person's take
		{ID: "p39w4a", State: "doing", Assignee: []string{"claude-opus-5-5"}, Path: "p/p39w4a.md", Model: "claude-opus-5-5"}, // the model is the name
		{ID: "e3vw30", State: "doing", Assignee: []string{"kestrel", "macaque"}, Path: "e/e3vw30.md"},                        // one live owner is enough
	}
	got := map[string]string{}
	live := LivenessIn([]string{repo}, time.Time{}, dir, filepath.Join(other, "ssb"))
	for _, f := range OwnerFindings(live, ts, now) {
		id, _ := IDOf(f.Path)
		got[id] = f.Problem
		if FindingCode(f.Problem) != "stale-owner" || f.Fix != "hug task set "+id+" --assignee NAME" {
			t.Errorf("%s: code %q fix %q", id, FindingCode(f.Problem), f.Fix)
		}
	}
	for _, id := range []string{"zhvkmj", "tb0dra", "bxqtkt", "jr55s8"} {
		if got[id] == "" {
			t.Errorf("%s not reported (its owner is gone); findings: %q", id, got)
		}
	}
	for _, id := range []string{"sortxx", "911j70", "t7wetj", "p39w4a", "e3vw30"} {
		if got[id] != "" {
			t.Errorf("%s reported, its owner is live or not a session: %s", id, got[id])
		}
	}
	if p := got["zhvkmj"]; p != "" && !strings.Contains(p, "since "+now.Add(-6*time.Hour-14*time.Minute).Local().Format("2006-01-02 15:04")) {
		t.Errorf("nautilus: the last sign of life is the journal's time: %s", p)
	}
	if p := got["bxqtkt"]; p != "" && !strings.Contains(p, "kestrel") {
		t.Errorf("kestrel: %s", p)
	}
}

// agh2h7: the recall hook's coordinator line asks Gone, which is
// stale-owner's own rule, so the two never disagree on who is gone; At is
// the same list as it stood at an earlier moment (board4 replays).
func TestLivenessGoneAndAt(t *testing.T) {
	dir := t.TempDir()
	repo := "/Users/jhf/Projects/hug"
	now := time.Date(2026, 11, 18, 14, 0, 0, 0, time.UTC)
	ms := func(at time.Time) string { return strconv.FormatInt(at.UnixMilli(), 10) }
	write := func(name string, start time.Time, status string, at time.Time, title string) {
		t.Helper()
		meta := `"working_dir":"` + repo + `","short_name":"` + name + `","status":"` + status + `"`
		p := filepath.Join(dir, "session_"+name+"_"+ms(start)+"_ab12.json")
		if err := os.WriteFile(p, []byte(`{"id":"x","title":"`+title+`","messages":[],`+meta+`}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, at, at); err != nil {
			t.Fatal(err)
		}
	}
	// wyvern: started 08:00, closed at 10:00 (4 h ago).
	write("wyvern", now.Add(-6*time.Hour), "Closed", now.Add(-4*time.Hour), "")
	// puppy: closed 20 minutes ago: not yet gone (GoneOwnerAfter).
	write("puppy", now.Add(-time.Hour), "Closed", now.Add(-20*time.Minute), "")
	// calf, told "You are kestrel", active a minute ago.
	write("calf", now.Add(-2*time.Hour), "Active", now.Add(-time.Minute), "You are kestrel, a worker on hug")
	live := LivenessIn([]string{repo}, time.Time{}, dir)
	for _, c := range []struct {
		name        string
		gone, known bool
	}{{"wyvern", true, true}, {"puppy", false, true}, {"kestrel", false, true}, {"calf", false, true}, {"jhf", false, false}} {
		if gone, known := live.Gone(c.name, now); gone != c.gone || known != c.known {
			t.Errorf("Gone(%s) = %v, %v; want %v, %v", c.name, gone, known, c.gone, c.known)
		}
		// The same answer as stale-owner on a doing task of that name.
		f := OwnerFindings(live, []Task{{ID: "a1b2c3", State: "doing", Assignee: []string{c.name}, Path: "a/a1b2c3.md", Fields: map[string]string{"doing_since": "2026-11-18T13:00"}}}, now)
		if (len(f) == 1) != c.gone {
			t.Errorf("%s: Gone says %v, stale-owner says %v", c.name, c.gone, f)
		}
	}
	// At 09:30 wyvern was still writing: live then, closed only later.
	then := now.Add(-4*time.Hour - 30*time.Minute)
	if gone, known := live.At(then).Gone("wyvern", then); gone || !known {
		t.Errorf("wyvern at 09:30: gone %v known %v, want live", gone, known)
	}
	// At 11:30 it had closed 1.5 h before: gone.
	later := now.Add(-2*time.Hour - 30*time.Minute)
	if gone, _ := live.At(later).Gone("wyvern", later); !gone {
		t.Error("wyvern at 11:30: not gone")
	}
	// Before puppy started, the list does not know it.
	if _, known := live.At(now.Add(-2*time.Hour)).Gone("puppy", now.Add(-2*time.Hour)); known {
		t.Error("puppy known before its session started")
	}
	// The alias holds in the past view too.
	if _, known := live.At(now).Gone("kestrel", now); !known {
		t.Error("kestrel alias lost in At")
	}
}

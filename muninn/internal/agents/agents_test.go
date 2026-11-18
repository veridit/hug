package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// Cases from the seid hand-check (2026-11-16): agents in agent context are
// found; animals, nouns and design names are not.
func TestFind(t *testing.T) {
	m := New([]string{"orangutan", "dragon", "bug", "scorpion", "tulip", "tiger", "lion", "fox", "badger"})
	for text, want := range map[string][]string{
		"TASK-1615 review5 verdict was reported to coordinator orangutan.":                       {"orangutan"},
		"Do not build while Dragon’s long chain dry-run is running.":                             {"dragon"},
		"reviewer scorpion was spawned with claude-opus-5-5 high effort":                         {"scorpion"},
		"For protocol mismatches, Tulip should stop before later gates.":                         {"tulip"},
		"Architect lion (session_lion_1787816218445_2dec1ab8d3de4f1e) assigned the rename work.": {"lion"},
		"The King corrected Dragon 1's frame and the packet is revised.":                         nil,
		"TASK-370 wave-1: 9e712622 (Dragon-A kill) landed.":                                      nil,
		"The campaign keeps independent bug scouting lanes.":                                     nil,
		"A tiger is a large cat.": nil,
		"Brovs-4835 tracks the possible production timestamp-offset bug.": nil,
		"**What**: Scout bug hunt on seid-lang 2026-08-17.":               nil,
		"Nothing. No bug. The architect caught it by reading.":            nil,
		"Architect authors dragon-task SPECS directly.":                   nil,
		"LSM engines: LevelDB, RocksDB, Pebble, Badger.":                  nil,
		"When a Brovs task is ambiguous, DM coordinator fox and wait.":    {"fox"},
		"Lane B (dolphin/Sol) claimed it; reviewer scorpion was spawned.": {"scorpion"},
		"DM fox only at defined milestones.":                              {"fox"},
		"Drill attempt 8 is comment 101-8000; coordinator is fox.":        {"fox"},
		"The interactive queue architect is the fox session.":             {"fox"},
		// Known false match, 1 of the 2 in the 50-item hand-check: a noun
		// right after a context word. The dream item it raises is judged by
		// an agent, so a miss costs one question, never a memory.
		"The emitted-C diff was a gate artifact, not a compiler lane bug.": {"bug"},
	} {
		if got := m.Find(text); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v, want %v", text, got, want)
		}
	}
}

// typq1e: a memory may name an agent with no context word of its own
// ("spider pushes", "via fox") when the transcript it came from uses that
// name as an agent. Cases from the CAPTURE4 outputs; the thing-words stay.
func TestFindIn(t *testing.T) {
	m := New([]string{"spider", "fox", "squid", "dragon", "crab", "bug"})
	transcript := "DM from coordinator spider: fox reviews. Cross-swarm DM from fox. operator squid failed. reviewer crab approved; lane bug fixed."
	for text, want := range map[string][]string{
		"In Brovs DevOps, spider pushes commits; agents commit locally.":      {"spider"},
		"Brovs owner rule (via fox): QA switches need no approval.":           {"fox"},
		"hug task take refused with 'taken by crab (qawmnb)'.":                {"crab"},
		"The fix waits for squid's P0 capture fix to land.":                   {"squid"},
		"A spider web on the server rack is not a defect.":                    nil,
		"The rack photo shows a house spider, not a defect.":                  nil,
		"Spider-2 is the design name of the second crawler.":                  nil,
		"The dragon chain dry-run is green.":                                  nil, // not in the transcript as an agent
		"The emitted-C diff was a gate artifact, not a compiler bug.":         nil, // "a compiler bug": the thing
		"Quartz deployment results are reported to coordinator spider daily.": {"spider"},
	} {
		if got := m.FindIn(text, transcript); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v, want %v", text, got, want)
		}
	}
	if got := m.FindIn("In Brovs DevOps, spider pushes commits.", ""); got != nil {
		t.Errorf("no transcript: %v, want the plain Find result (nil)", got)
	}
}

func TestMeant(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 11, d, 12, 0, 0, 0, time.UTC) }
	// orangutan in September (1-3) and again today (10-11): a memory from
	// the 2nd means the first one, one from the 5th the newest before it.
	ss := []Session{{Path: "a", Start: day(1), Last: day(3)}, {Path: "b", Start: day(10), Last: day(11)}}
	for at, want := range map[int]string{2: "a", 5: "a", 10: "b"} {
		if got := Meant(ss, day(at)); len(got) != 1 || got[0].Path != want {
			t.Errorf("memory on day %d: %v, want %s", at, got, want)
		}
	}
	if got := Meant(ss, day(0)); got != nil {
		t.Errorf("before any session: %v", got)
	}
}

func TestNames(t *testing.T) {
	h := t.TempDir()
	os.MkdirAll(filepath.Join(h, "sessions"), 0o755)
	for _, f := range []string{"session_crab_1793783281662_44ffaee927a8a9d5.json", "session_crab_1_ab.json", "session_vole_2_cd.journal.jsonl", "notes.txt"} {
		os.WriteFile(filepath.Join(h, "sessions", f), nil, 0o644)
	}
	if got := Names([]string{h, filepath.Join(h, "missing")}); !reflect.DeepEqual(got, []string{"crab"}) {
		t.Fatalf("Names = %v", got)
	}
}

// rnpdnm attempt 2 (a): NamesIn is the agents whose sessions worked in
// one of roots (working_dir is a root or inside one), read from the end of
// each session file; another repository's agents are not this team.
func TestNamesIn(t *testing.T) {
	h := t.TempDir()
	os.MkdirAll(filepath.Join(h, "sessions"), 0o755)
	repo := filepath.Join(h, "repo")
	os.MkdirAll(filepath.Join(repo, "sub"), 0o755)
	write := func(name, wd string) {
		body := `{"id":"x","messages":[],"working_dir":` + quoteJSON(wd) + `,"short_name":"x","status":"Active"}`
		os.WriteFile(filepath.Join(h, "sessions", name), []byte(body), 0o644)
	}
	write("session_zarf_1_ab.json", repo)
	write("session_quillback_2_cd.json", filepath.Join(repo, "sub"))
	write("session_wyvern_3_ef.json", filepath.Join(h, "elsewhere"))
	os.WriteFile(filepath.Join(h, "sessions", "session_vole_4_aa.json"), []byte(`{"messages":[]}`), 0o644)
	if got := NamesIn([]string{h}, []string{repo}); !reflect.DeepEqual(got, []string{"quillback", "zarf"}) {
		t.Fatalf("NamesIn = %v", got)
	}
}

func quoteJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/ingest"
)

// rnpdnm check 1 (B): the gate holds a line naming a home folder, another
// project or client, or the person, and a preference or correction; the
// rest of the project scope pushes. Crab: whole words still catch
// possessives and the ASCII fold ("Jørgen's", "Jorgen", "jhf:").
func TestGateHoldsByRule(t *testing.T) {
	g := &Gate{Terms: []string{"seid-lang", "brovs", "jhf", "Jørgen H. Fjeld", "Jørgen", "search", "the user"}, Own: []string{"hug"}}
	cases := []struct {
		text, category, want string
	}{
		{"Builds run under /Users/someone/Projects with nice", "fact", HeldHome},
		{"Keep the scratch file at ~/notes.md", "fact", HeldHome},
		{`Windows checkout lives at C:\Users\x\src`, "fact", HeldHome},
		{"The server keeps it in /home/deploy/app", "fact", HeldHome},
		{"seid-lang uses K3 for memory safety", "fact", HeldName},
		{"Ship it the Brovs way", "decision", HeldName},
		{"Jørgen's rule: one ticket at a time", "decision", HeldName},
		{"Jorgen wants short replies", "fact", HeldName},
		{"jhf: confirm on the board", "fact", HeldName},
		{"Answer the user in Norwegian", "fact", HeldName},
		{"Use tabs in Go files", "preference", HeldPreference},
		{"Not that flag, the other one", "correction", HeldPreference},
		// Pushes: no marker, whole words only, own names, home-like paths.
		{"hug task lint never writes", "fact", ""},
		{"Recall research uses BM25 scale 2.0", "fact", ""},
		{"jhfx is not a name", "fact", ""},
		{"Store files under .hug/memory in the repo", "decision", ""},
		{"Tests set HOME to a temp dir, see internal/home/x.go", "fact", ""},
		{"Searching is keyword plus semantic", "fact", ""},
	}
	for _, c := range cases {
		got, _ := g.Hold(ingest.NativeMemory{ID: "mem_1", Content: c.text, Category: c.category}, "mem_1")
		if got != c.want {
			t.Errorf("%q (%s): held %q, want %q", c.text, c.category, got, c.want)
		}
	}
	// A term equal to the project's own name never holds its memories.
	if got, _ := (&Gate{Terms: []string{"hug"}, Own: []string{"hug"}}).Hold(ingest.NativeMemory{Content: "hug syncs memories"}, ""); got != "" {
		t.Errorf("own name held: %q", got)
	}
	// A local exception releases one memory, by full or short id or line id.
	rel := &Gate{Terms: []string{"brovs"}, Release: []string{"abc123"}}
	for _, m := range []ingest.NativeMemory{{ID: "mem_x", SID: "abc123", Content: "brovs"}, {ID: "abc123", Content: "brovs"}} {
		if got, _ := rel.Hold(m, ""); got != "" {
			t.Errorf("released %+v held: %q", m, got)
		}
	}
	if got, _ := rel.Hold(ingest.NativeMemory{ID: "mem_y", Content: "brovs"}, "abc123"); got != "" {
		t.Errorf("released line id held: %q", got)
	}
	// No gate (nil) holds nothing: Sync without a gate is unchanged.
	var none *Gate
	if got, _ := none.Hold(ingest.NativeMemory{Content: "/Users/x"}, ""); got != "" {
		t.Errorf("nil gate held %q", got)
	}
}

// The gate at sync: held lines never reach the file, the rest do, and a
// line that was in the file before (pushed before the gate) drops from the
// tip and is counted as already in the remote's history (crab point 3).
func TestSyncGateWritesOnlyWhatPasses(t *testing.T) {
	dir := t.TempDir()
	p := Project{ID: "rc-test", Label: "hug", Root: dir}
	native := filepath.Join(dir, "native.json")
	write := func(text, cat string) {
		if _, _, err := ingest.Remember(native, text, cat, p.Scope(), nil); err != nil {
			t.Fatal(err)
		}
	}
	write("Synthetic project fact that pushes", "fact")
	write("Synthetic: Jørgen confirms on his board", "fact")
	if _, err := Sync(p, native, false); err != nil { // before the gate: both pushed
		t.Fatal(err)
	}
	write("Synthetic preference for short replies", "preference")
	write("Synthetic path /Users/someone/x", "fact")
	write("Synthetic note about seid-lang", "fact")
	g := &Gate{Terms: []string{"Jørgen", "seid-lang"}, Own: []string{"hug"}}
	st, err := SyncGated(p, native, false, g)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p.File())
	file := string(b)
	if !strings.Contains(file, "Synthetic project fact that pushes") {
		t.Fatalf("passing line missing: %s", file)
	}
	for _, held := range []string{"Jørgen", "short replies", "/Users/", "seid-lang"} {
		if strings.Contains(file, held) {
			t.Errorf("held line %q in the file: %s", held, file)
		}
	}
	if st.Written != 1 || st.HeldTotal() != 4 || st.Held[HeldName] != 2 || st.Held[HeldHome] != 1 || st.Held[HeldPreference] != 1 {
		t.Fatalf("stats %+v", st)
	}
	if st.HeldPushed != 1 {
		t.Fatalf("HeldPushed=%d, want 1 (the Jørgen line was pushed before the gate)", st.HeldPushed)
	}
	// The held list is read-only and agrees with the sync.
	before, _ := os.ReadFile(native)
	held, err := HeldLines(p, native, g)
	if err != nil || len(held) != 4 {
		t.Fatalf("HeldLines: %v %+v", err, held)
	}
	after, _ := os.ReadFile(native)
	if string(before) != string(after) {
		t.Fatal("HeldLines wrote the store")
	}
	pushed := 0
	for _, h := range held {
		if h.Pushed {
			pushed++
		}
	}
	if pushed != 1 {
		t.Fatalf("held list marks %d as already pushed, want 1: %+v", pushed, held)
	}
}

func TestPeopleTerms(t *testing.T) {
	got := PeopleTerms([]byte("layout = 3\n\n[people.jhf]\nname = \"Jørgen H. Fjeld\"\nemail = [\"a@b\"]\nalias = [\"jorgen\", \"JHF\"]\n\n[board]\ncoordinator = \"crab\"\n"))
	want := "jhf|Jørgen H. Fjeld|Jørgen|jorgen|JHF"
	if strings.Join(got, "|") != want {
		t.Fatalf("got %q, want %q", strings.Join(got, "|"), want)
	}
}

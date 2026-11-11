package recall

import (
	"strings"
	"testing"
	"time"
)

func TestSaveCue(t *testing.T) {
	fire := []string{
		"From now on, run make lint before every commit.",
		"No, that's wrong, the staging port is 8443.",
		"Never use Chrome for browser automation here.",
		"I prefer when you keep answers short.",
		"We decided to drop the legacy importer.",
		"Fra nå av skal du alltid svare på norsk.",
		"Dette er feil, vi bruker ikke Docker lenger.",
		"That's outdated, the build moved to GitHub Actions.",
	}
	quiet := []string{
		"thanks",
		"ok, continue",
		"can you check the CI",
		"should we always run the tests first?",
		"walk down the hallways and never look back", // "never look" is not a listed verb
		"refactor this function",
		"[muninn recall] From now on use tabs",
		"",
	}
	for _, m := range fire {
		if SaveCue(m) == "" {
			t.Errorf("should cue: %q", m)
		}
	}
	for _, m := range quiet {
		if c := SaveCue(m); c != "" {
			t.Errorf("should stay quiet: %q -> %q", m, c)
		}
	}
}

func TestRenderCarriesIDsAndFixRule(t *testing.T) {
	hits := []Hit{{SID: "k7m2qx", Key: "mem_1a2b", Source: "native", Text: "Uses tabs", Updated: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}, {SID: "abc123", Key: "abc-123", Source: "jcode", Text: "Port 8443"}}
	b := Render(hits, true)
	short := Render(hits, false)
	if short != b || !strings.Contains(short, ".0:2026-03-01 .1:k7m2qx .2:`Uses tabs`") || !IsOwnOutput(short) {
		t.Fatalf("short block wrong:\n%s", short)
	}
	// 2026-11-10: header chosen by a 5-agent test (F1, 38/40): it must say the
	// notes are not the user's, newer wins, and how to fix one.
	for _, want := range []string{".0:2026-03-01 .1:k7m2qx .2:`Uses tabs`", ".0: .1:abc123 .2:`Port 8443`", "AI memory", "not the user", "newer wins", "memory_forget <id>", "memory_remember"} {
		if !contains(b, want) {
			t.Errorf("block missing %q:\n%s", want, b)
		}
	}
	if !IsOwnOutput(b) {
		t.Error("block must still be recognised as muninn's own output")
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

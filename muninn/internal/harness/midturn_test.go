package harness

import (
	"os"
	"path/filepath"
	"testing"
)

// The mid-turn reader (x8zzzn) takes jcode's turn id from each line's meta,
// skips messages jcode wrote with a display role, leaves a line jcode is
// still writing for the next read, and starts over when the journal shrank.
func TestReadJcodeTail(t *testing.T) {
	dir := t.TempDir()
	sid := "session_tail_1"
	path := filepath.Join(dir, sid+".journal.jsonl")
	line := func(turn, msg string) string {
		return `{"meta":{"model_usage_turn_id":"` + sid + `:` + turn + `"},"append_messages":[` + msg + `]}` + "\n"
	}
	user := func(id, text string) string {
		return `{"id":"` + id + `","role":"user","content":[{"type":"text","text":"` + text + `"}]}`
	}
	start := line("m1", user("m1", "first"))
	body := start +
		line("m1", `{"role":"assistant","content":[{"type":"tool_use"}]}`) +
		line("m1", user("m2", "pasted")) +
		line("m1", `{"id":"m3","role":"user","display_role":"system","content":[{"type":"text","text":"swarm dm"}]}`)
	partial := `{"meta":{"model_usage_turn_id":"` + sid + `:m1"},"append_messages":[` + user("m4", "half")
	if err := os.WriteFile(path, []byte(body+partial), 0o600); err != nil {
		t.Fatal(err)
	}
	turn, end, ok := JcodeTurnOf(dir, sid, "m1")
	if !ok || turn != "m1" || end != int64(len(start)) {
		t.Fatalf("JcodeTurnOf = %q %d %v, want m1 %d true", turn, end, ok, len(start))
	}
	tail, err := ReadJcodeTail(dir, sid, end)
	if err != nil {
		t.Fatal(err)
	}
	if got := tail.Pasted("m1"); len(got) != 1 || got[0].ID != "m2" || got[0].Text != "pasted" {
		t.Fatalf("Pasted = %+v, want only m2", got)
	}
	if tail.End != int64(len(body)) {
		t.Fatalf("End = %d, want %d: the half-written line is read next time", tail.End, len(body))
	}
	if nt := tail.NewTurn("m1"); nt != "" {
		t.Fatalf("NewTurn = %q in the same turn", nt)
	}
	// jcode finishes the line and starts a new turn.
	if err := os.WriteFile(path, []byte(body+partial+"]}\n"+line("m5", user("m5", "next turn"))), 0o600); err != nil {
		t.Fatal(err)
	}
	tail, _ = ReadJcodeTail(dir, sid, tail.End)
	if got := tail.Pasted("m1"); len(got) != 1 || got[0].ID != "m4" {
		t.Fatalf("after the line completed: Pasted = %+v, want m4", got)
	}
	if nt := tail.NewTurn("m1"); nt != "m5" {
		t.Fatalf("NewTurn = %q, want m5", nt)
	}
	// Compaction rewrote the journal shorter than the saved offset.
	if err := os.WriteFile(path, []byte(start), 0o600); err != nil {
		t.Fatal(err)
	}
	tail, err = ReadJcodeTail(dir, sid, 1<<20)
	if err != nil || tail.End != int64(len(start)) || len(tail.Msgs) != 1 {
		t.Fatalf("after a rewrite: %+v err=%v", tail, err)
	}
}

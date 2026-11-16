package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"

	"github.com/veridit/hug/muninn/internal/app"
)

// check 4 (ez50rb): a memory naming a swarm agent becomes one dream item
// naming the ticket that agent worked on; nothing is deleted, and a memory
// using the same word as an animal is left alone.
func TestAgentNamesBecomeDreamItems(t *testing.T) {
	f := scopetest.New(t)
	ctx := context.Background()
	sessions := filepath.Join(f.Root, "alpha", ".jcode", "sessions")
	if err := os.MkdirAll(sessions, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessions, "session_orangutan_1787738929412_b29823251e479ae4.json"),
		[]byte(`{"messages":[{"content":"hug task take ab12cd --model x; TASK-1615 review, TASK-1615 again"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// orangutan stopped two days ago; vole is running (its file is fresh).
	old := time.Now().Add(-48 * time.Hour)
	os.Chtimes(filepath.Join(sessions, "session_orangutan_1787738929412_b29823251e479ae4.json"), old, old)
	vole := filepath.Join(sessions, "session_vole_1787738929999_aa.json")
	if err := os.WriteFile(vole, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	for id, text := range map[string]string{
		"mem_ag1": "TASK-1615 review verdict was reported to coordinator orangutan.",
		"mem_ag2": "An orangutan is a great ape that lives in Borneo and Sumatra.",
		"mem_ag3": "Report hug build results to coordinator vole.",
	} {
		mems = append(mems, ingest.NativeMemory{ID: id, Content: text, Category: "fact", Scope: "project:matrix-project", Active: true, Strength: 1, Created: time.Now(), Updated: time.Now()})
	}
	if err := ingest.SaveNative(app.NativePath(), mems); err != nil {
		t.Fatal(err)
	}
	if _, err := f.App.Reindex(ctx); err != nil {
		t.Fatal(err)
	}
	sid := map[string]string{}
	mems, _ = ingest.LoadNative(app.NativePath())
	for _, m := range mems {
		sid[m.ID] = m.SID
	}
	before := 0
	f.App.DB.QueryRow(`SELECT count(*) FROM units WHERE active=1`).Scan(&before)
	for i := 0; i < 2; i++ {
		if _, err := f.App.Dream(ctx, false); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	var instr string
	f.App.DB.QueryRow(`SELECT count(*), coalesce(max(instruction),'') FROM dream_items WHERE kind='agent' AND sids=?`, sid["mem_ag1"]).Scan(&n, &instr)
	if n != 1 || !strings.Contains(instr, "names orangutan") || !strings.Contains(instr, "TASK-1615") {
		t.Fatalf("agent item: n=%d %q", n, instr)
	}
	f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='agent' AND sids=?`, sid["mem_ag2"]).Scan(&n)
	if n != 0 {
		t.Fatal("the animal must not be taken for the agent")
	}
	f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='agent' AND sids=?`, sid["mem_ag3"]).Scan(&n)
	if n != 0 {
		t.Fatal("a memory naming a running agent waits until it stops")
	}
	// vole stops: the next pass rescans from the waiting memory and queues it.
	os.Chtimes(vole, old, old)
	if _, err := f.App.Dream(ctx, false); err != nil {
		t.Fatal(err)
	}
	f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='agent' AND sids=?`, sid["mem_ag3"]).Scan(&n)
	if n != 1 {
		t.Fatalf("stopped vole: %d items, want 1", n)
	}
	f.App.DB.QueryRow(`SELECT count(*) FROM dream_items WHERE kind='agent' AND sids=?`, sid["mem_ag1"]).Scan(&n)
	if n != 1 {
		t.Fatalf("orangutan item offered %d times, want once", n)
	}
	after := 0
	f.App.DB.QueryRow(`SELECT count(*) FROM units WHERE active=1`).Scan(&after)
	if after != before {
		t.Fatalf("nothing is deleted automatically: %d -> %d", before, after)
	}
}

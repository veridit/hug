package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/veridit/hug/internal/timing"
	_ "modernc.org/sqlite"
)

// heldStore makes a memory store that cannot open: a database with none of
// hug's tables, write-locked by the test, so store.Open's schema waits for
// the lock (busy_timeout, 30 s) as it did after a release while another hug
// indexed (vrx1ja). release lets it open.
func heldStore(t *testing.T, hugHome string) (release func()) {
	t.Helper()
	dir := filepath.Join(hugHome, "data")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.Join(dir, "muninn.db"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	c, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`PRAGMA journal_mode=WAL`, `CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`, `BEGIN IMMEDIATE`, `INSERT INTO meta VALUES('held', 'by the test')`} {
		if _, err := c.ExecContext(ctx, q); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	done := false
	release = func() {
		if done {
			return
		}
		done = true
		c.ExecContext(ctx, `ROLLBACK`)
		c.Close()
		db.Close()
	}
	t.Cleanup(release)
	return release
}

// slowEmbedder is an Ollama that answers its probe only after a while, set
// in a config.json under hugHome.
func slowEmbedder(t *testing.T, hugHome string, delay time.Duration) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
		w.Write([]byte(`{"models":[]}`))
	}))
	t.Cleanup(srv.Close)
	cfg := filepath.Join(hugHome, "config")
	if err := os.MkdirAll(cfg, 0o700); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"embedder": "ollama", "ollama_url": srv.URL})
	if err := os.WriteFile(filepath.Join(cfg, "config.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
}

// mcpTestRepo is a throwaway home, memory home and git repository, the
// working directory for the test.
func mcpTestRepo(t *testing.T) (hugHome string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	hugHome = t.TempDir()
	t.Setenv("HUG_HOME", hugHome)
	t.Setenv("JCODE_HOME", "")
	t.Setenv("HUG_SYNC", "off")
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, out)
		}
	}
	old, _ := os.Getwd()
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return hugHome
}

type mcpStart struct {
	s       *sdk.Server
	closeFn func()
	err     error
}

// TestMCPAnswersBeforeMemoryOpens (vrx1ja check 1): with the memory
// database write-locked and the embedder slow, hug mcp answers initialize
// within a second, tools/list has the task tools (and the memory tools,
// same names as when memory is up), and a memory tool says memory is still
// opening, with a fix: line, instead of hanging. Once the lock goes, memory
// opens and the same tool works.
func TestMCPAnswersBeforeMemoryOpens(t *testing.T) {
	timing.Budget(t)
	hugHome := mcpTestRepo(t)
	slowEmbedder(t, hugHome, 3*time.Second)
	release := heldStore(t, hugHome)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	started := make(chan mcpStart, 1)
	go func() {
		s, closeFn, err := newMCPServer(ctx, "")
		started <- mcpStart{s, closeFn, err}
	}()
	var st mcpStart
	select {
	case st = <-started:
	case <-time.After(time.Second):
		// Let the blocked open finish before the temp dirs go.
		release()
		st = <-started
		if st.closeFn != nil {
			st.closeFn()
		}
		t.Fatalf("hug mcp was not ready to answer initialize after 1 s: it waited %s for memory", time.Since(start).Round(time.Millisecond))
	}
	if st.err != nil {
		t.Fatal(st.err)
	}
	defer st.closeFn()
	srvT, cliT := sdk.NewInMemoryTransports()
	if _, err := st.s.Connect(ctx, srvT, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	if d := time.Since(start); d > time.Second {
		t.Fatalf("initialize answered after %s, want under 1 s", d.Round(time.Millisecond))
	}
	if !strings.Contains(cs.InitializeResult().Instructions, "task_next") {
		t.Errorf("instructions do not name the task tools:\n%s", cs.InitializeResult().Instructions)
	}

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("tools/list answered %s after start, want it without waiting for memory", d.Round(time.Millisecond))
	}
	var names []string
	for _, tl := range tools.Tools {
		names = append(names, tl.Name)
	}
	opening := toolSchemas(t, tools)
	for _, want := range []string{"task_list", "task_new", "task_next", "task_show", "memory_search", "memory_remember", "memory_stats"} {
		if !slices.Contains(names, want) {
			t.Errorf("tools/list while memory opens lacks %s: %v", want, names)
		}
	}

	callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
	r, err := cs.CallTool(callCtx, &sdk.CallToolParams{Name: "memory_stats", Arguments: map[string]any{}})
	callCancel()
	if err != nil {
		t.Fatalf("memory_stats while memory opens: %v (it must answer, not hang)", err)
	}
	out := toolText(r)
	if !r.IsError || !strings.Contains(out, "memory is still opening") || !strings.Contains(out, "\nfix: ") {
		t.Errorf("memory_stats while memory opens: isError=%v, want an error saying memory is still opening with a fix: line:\n%s", r.IsError, out)
	}
	if r, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: "task_types", Arguments: map[string]any{}}); err != nil || r.IsError {
		t.Errorf("task_types while memory opens: %v %s", err, toolText(r))
	}

	release()
	deadline := time.Now().Add(60 * time.Second)
	for {
		r, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: "memory_stats", Arguments: map[string]any{}})
		if err != nil {
			t.Fatal(err)
		}
		if !r.IsError {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("memory never became ready after the lock went: %s", toolText(r))
		}
		time.Sleep(200 * time.Millisecond)
	}
	// Same tools, same schemas, once memory is up (vrx1ja: jcode and Claude
	// see no difference but the timing).
	up, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := toolSchemas(t, up); got != opening {
		t.Errorf("tools/list changed once memory opened:\nwhile opening: %s\nonce up:       %s", opening, got)
	}
}

// toolSchemas is every tool's name, input and output schema, as one string.
func toolSchemas(t *testing.T, r *sdk.ListToolsResult) string {
	t.Helper()
	b, err := json.Marshal(r.Tools)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestMCPMemoryOpensAfterALongLock: a write lock held past one busy_timeout
// (an index pass holds it for minutes) leaves memory opening, not
// unavailable for the session; it opens once the lock goes (vrx1ja).
func TestMCPMemoryOpensAfterALongLock(t *testing.T) {
	if testing.Short() {
		t.Skip("holds the store past busy_timeout (30 s)")
	}
	hugHome := mcpTestRepo(t)
	t.Setenv("HUG_NO_EMBED", "1")
	release := heldStore(t, hugHome)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s, closeFn, err := newMCPServer(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()
	srvT, cliT := sdk.NewInMemoryTransports()
	if _, err := s.Connect(ctx, srvT, nil); err != nil {
		t.Fatal(err)
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, cliT, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	// Past one busy_timeout (store.BusyTimeout, 30 s): the first open has
	// given up by now.
	time.Sleep(33 * time.Second)
	r, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: "memory_stats", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if out := toolText(r); !r.IsError || !strings.Contains(out, "memory is still opening") {
		t.Fatalf("memory_stats with the lock held 33 s: want still opening, got isError=%v:\n%s", r.IsError, out)
	}
	release()
	deadline := time.Now().Add(30 * time.Second)
	for {
		r, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: "memory_stats", Arguments: map[string]any{}})
		if err != nil {
			t.Fatal(err)
		}
		if !r.IsError {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("memory did not open after a lock held past busy_timeout: %s", toolText(r))
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func toolText(r *sdk.CallToolResult) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

package run

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/veridit/hug/internal/timing"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/store"
)

// t0qap9 check 4: a remember from another process during a full reindex
// succeeds within 5 s. `hug mem index` runs in one child process against
// a fake Ollama whose batch calls are slow, so the pass takes several
// seconds; once it is embedding, `hug mem remember` runs in a second child.
// Before t0qap9 the pass held the write turn from start to end, so the
// remember waited for all of it.
func TestRememberDuringFullReindexSucceedsWithin5s(t *testing.T) {
	if os.Getenv("HUG_CHILD_ARGS") != "" {
		var args []string
		json.Unmarshal([]byte(os.Getenv("HUG_CHILD_ARGS")), &args)
		os.Exit(Main(args))
	}
	if testing.Short() {
		t.Skip("spawns 2 processes and a multi-second index pass")
	}
	timing.Budget(t)
	home := memHome(t)
	t.Setenv("HUG_NO_EMBED", "")
	var batches atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.Write([]byte(`{"models":[]}`))
			return
		}
		var req struct {
			Input []string `json:"input"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Input) > 1 {
			batches.Add(1)
			time.Sleep(120 * time.Millisecond) // the pass's batches: slow
		}
		out := make([][]float32, len(req.Input))
		for i, text := range req.Input {
			v := make([]float32, 32)
			for _, word := range strings.Fields(strings.ToLower(text)) {
				h := fnv.New32a()
				h.Write([]byte(word))
				v[h.Sum32()%32]++
			}
			v[0] += 0.01
			out[i] = v
		}
		json.NewEncoder(w).Encode(map[string]any{"embeddings": out})
	}))
	defer srv.Close()
	cfg, _ := config.Load()
	cfg.Embedder, cfg.OllamaURL, cfg.EmbedModel = "ollama", srv.URL, "fake-slow"
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	// 2,000 memories: about 63 slow batches, 7.5 s of embedding.
	const n = 2000
	words := strings.Fields("quartz harbor ledger velvet cobalt meadow lantern orbit saffron tundra beacon marble falcon prism willow ember garnet canyon")
	now := time.Now().UTC()
	mems := make([]ingest.NativeMemory, n)
	for i := range mems {
		var sb strings.Builder
		for j := 0; j < 8; j++ {
			sb.WriteString(words[(i*7+j*j+i/len(words))%len(words)])
			sb.WriteByte(' ')
		}
		fmt.Fprintf(&sb, "note %d", i)
		mems[i] = ingest.NativeMemory{ID: fmt.Sprintf("mem_bulk_%04d", i), Content: sb.String(), Category: "fact", Scope: "global", Active: true, Created: now, Updated: now}
	}
	if err := ingest.SaveNative(app.NativePath(), mems); err != nil {
		t.Fatal(err)
	}
	child := func(args ...string) *exec.Cmd {
		b, _ := json.Marshal(args)
		c := exec.Command(os.Args[0], "-test.run=^TestRememberDuringFullReindexSucceedsWithin5s$")
		c.Env = append(os.Environ(), "HUG_CHILD_ARGS="+string(b), "GOMAXPROCS=1")
		return c
	}
	idx := child("index")
	var idxOut strings.Builder
	idx.Stdout, idx.Stderr = &idxOut, &idxOut
	t0 := time.Now()
	if err := idx.Start(); err != nil {
		t.Fatal(err)
	}
	indexDone := make(chan error, 1)
	go func() { indexDone <- idx.Wait() }()
	// Remember, one after another, from the pass's start until it ends:
	// its sources, embedding and neighbour steps each get some.
	var took []time.Duration
	var outs []string
	var during []int64 // slow batches seen when each remember began
	running := true
	for i := 0; running; i++ {
		seen := batches.Load()
		r0 := time.Now()
		out, err := child("remember", "--new-reason", "a distinct test memory", fmt.Sprintf("t0qap9 kestrel firmware handshake rule number %d during a reindex", i)).CombinedOutput()
		d := time.Since(r0)
		if err != nil {
			t.Fatalf("remember %d during the reindex failed after %s: %v: %s", i, d, err, out)
		}
		outs = append(outs, strings.TrimSpace(string(out)))
		select {
		case err := <-indexDone:
			if err != nil {
				t.Fatalf("hug mem index: %v: %s", err, idxOut.String())
			}
			running = false
			t.Logf("CHECK4 the remember that saw the pass end took %s", d.Round(time.Millisecond))
		default:
			took, during = append(took, d), append(during, seen)
		}
		if time.Since(t0) > 3*time.Minute {
			t.Fatal("hug mem index ran over 3 minutes")
		}
	}
	pass := time.Since(t0)
	worst := time.Duration(0)
	for _, d := range took {
		worst = max(worst, d)
	}
	t.Logf("CHECK4 %d remembers finished during a %s full reindex; slowest %s; batches seen at their starts %v; times %v",
		len(took), pass.Round(time.Millisecond), worst.Round(time.Millisecond), during, took)
	if worst > 5*time.Second {
		t.Fatalf("a remember took %s during a full reindex, want at most 5 s", worst)
	}
	// Some remembers ran before the first slow batch (sources), some while
	// the pass embedded, and the pass itself ran long enough to prove it.
	if len(took) < 3 || during[0] != 0 || during[len(during)-1] < 3 {
		t.Fatalf("remembers did not span the pass: batches seen %v", during)
	}
	// The remembered memory is indexed, with its vector, and a pass finds
	// nothing left to do.
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var units, vecs, queued int
	db.QueryRow(`SELECT COUNT(*) FROM units u JOIN embeddings e ON e.content_hash=u.content_hash WHERE u.body LIKE '%kestrel firmware handshake%'`).Scan(&vecs)
	db.QueryRow(`SELECT COUNT(*) FROM units WHERE active=1`).Scan(&units)
	db.QueryRow(`SELECT (SELECT COUNT(*) FROM index_pending) + (SELECT COUNT(*) FROM index_deltas)`).Scan(&queued)
	// Each remember stored a new memory or reinforced one stored before.
	stored, reinforced := 0, 0
	for _, o := range outs {
		switch {
		case strings.HasPrefix(o, "remembered "):
			stored++
		case strings.HasPrefix(o, "reinforced "):
			reinforced++
		default:
			t.Fatalf("remember printed neither: %s", o)
		}
	}
	if units != n+stored || vecs != stored || queued != 0 {
		t.Fatalf("after the reindex: %d active units (want %d), %d remembered memories with a vector (want %d), %d queued index rows (want 0)", units, n+stored, vecs, stored, queued)
	}
	t.Logf("CHECK4 index after the pass: %d active units = %d + %d remembered (%d remembers reinforced one), each with its vector, 0 queued index rows", units, n, stored, reinforced)
	_ = home
}

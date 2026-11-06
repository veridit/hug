package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/embed"
	"github.com/veridit/muninn/internal/index"
	"github.com/veridit/muninn/internal/ingest"
	"github.com/veridit/muninn/internal/search"
	"github.com/veridit/muninn/internal/store"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type round struct {
	Key, Session, Date, Body string
	Evidence                 bool
}
type question struct {
	ID, Question, Date string
	Sessions           []string
	Rounds             []round
}

// Disk-caching only saves repeated indexing work. Query calls are not cached.
type diskEmbed struct {
	embed.Embedder
	Dir string
}

func (d diskEmbed) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	var missing []string
	var positions []int
	for i, s := range texts {
		h := sha256.Sum256([]byte(d.Model() + "\x00" + s))
		p := filepath.Join(d.Dir, hex.EncodeToString(h[:]))
		if b, e := os.ReadFile(p); e == nil {
			out[i] = embed.DecodeF32(b)
		} else {
			missing = append(missing, s)
			positions = append(positions, i)
		}
	}
	if len(missing) > 0 {
		v, e := d.Embedder.Embed(ctx, missing)
		if e != nil {
			return nil, e
		}
		if len(v) != len(missing) {
			return nil, fmt.Errorf("embedding count mismatch")
		}
		for j, i := range positions {
			out[i] = v[j]
			h := sha256.Sum256([]byte(d.Model() + "\x00" + texts[i]))
			if e = os.WriteFile(filepath.Join(d.Dir, hex.EncodeToString(h[:])), embed.EncodeF32(v[j]), 0600); e != nil {
				return nil, e
			}
		}
	}
	return out, nil
}
func dateValue(s string) time.Time { t, _ := time.Parse("2006/01/02 (Mon) 15:04", s); return t }
func keyphrases(s string) string {
	counts := map[string]int{}
	for _, w := range search.QueryWords(s) {
		if len(w) > 3 {
			counts[w]++
		}
	}
	var words []string
	for w := range counts {
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool {
		if counts[words[i]] == counts[words[j]] {
			return words[i] < words[j]
		}
		return counts[words[i]] > counts[words[j]]
	})
	if len(words) > 12 {
		words = words[:12]
	}
	return strings.Join(words, " ")
}
func longmem(input, dir string) {
	must(os.MkdirAll(dir, 0700))
	must(os.MkdirAll(filepath.Join(dir, "vectors"), 0700))
	data, err := os.ReadFile(input)
	must(err)
	var qs []question
	must(json.Unmarshal(data, &qs))
	cfg := config.Defaults()
	em := embed.NewOllama(cfg.OllamaURL, cfg.EmbedModel, 0)
	ctx := context.Background()
	if !em.Available(ctx) {
		panic("Ollama unavailable")
	}
	cache := diskEmbed{em, filepath.Join(dir, "vectors")}
	variants := []string{"baseline", "retrieval", "fts", "semantic", "no-community", "no-reserve", "no-gates", "no-semantic-gate", "no-relative-gate", "heuristic", "recency", "normalised-time", "union"}
	for n, q := range qs {
		path := filepath.Join(dir, q.ID)
		must(os.MkdirAll(path, 0700))
		must(os.Setenv("MUNINN_HOME", path))
		if _, err := os.Stat(filepath.Join(path, "done")); err == nil {
			continue
		}
		db, err := store.Open(filepath.Join(path, "muninn.db"))
		must(err)
		native := filepath.Join(path, "memories.json")
		var mems []ingest.NativeMemory
		for _, r := range q.Rounds {
			mems = append(mems, ingest.NativeMemory{ID: r.Key, Content: r.Body, Scope: "benchmark", Active: true, Created: dateValue(r.Date), Updated: dateValue(r.Date), Origin: "longmemeval:" + r.Key})
		}
		must(ingest.SaveNative(native, mems))
		ix := index.Indexer{DB: db, Embedder: cache}
		st, err := ix.Run(ctx, index.Options{NativePath: native})
		must(err)
		if st.EmbedErr != nil {
			panic("benchmark embedding failed")
		}
		fmt.Fprintf(os.Stderr, "question=%s indexed=%d embedded=%d elapsed=%s\n", q.ID, len(mems), st.Embedded, st.Elapsed)
		s := &search.Searcher{DB: db, Embedder: em}
		f, err := os.Create(filepath.Join(path, "results.jsonl"))
		must(err)
		enc := json.NewEncoder(f)
		for _, v := range variants {
			o := options(search.Options{Limit: 10, MinScore: 6, SemanticWeight: 10, SemanticFloor: .5, SemanticSlots: 3, Scopes: []string{"benchmark"}}, v)
			if v == "retrieval" {
				o.MinScore = 0
			}
			o.RecencyNow = dateValue(q.Date)
			query := q.Question
			if v == "heuristic" {
				query = focused(query)
			}
			t := time.Now()
			rs, err := s.Search(ctx, query, o)
			must(err)
			if v != "retrieval" && v != "no-gates" && v != "union" {
				rs = benchmarkGatesFor(rs, v)
			}
			must(enc.Encode(map[string]any{"id": q.ID, "variant": v, "results": rs, "latency_ms": float64(time.Since(t).Microseconds()) / 1000}))
		}
		// Source-backed keyword augmentation is deliberately not called LLM fact extraction.
		baselineCopy := filepath.Join(path, "baseline.db")
		if _, err := os.Stat(baselineCopy); os.IsNotExist(err) {
			_, err = db.Exec(`VACUUM INTO ?`, baselineCopy)
			must(err)
		}
		for i := range mems {
			mems[i].Title = keyphrases(mems[i].Content)
		}
		must(ingest.SaveNative(native, mems))
		st, err = ix.Run(ctx, index.Options{NativePath: native})
		must(err)
		if st.EmbedErr != nil {
			panic("key augmentation embedding failed")
		}
		t := time.Now()
		rs, err := s.Search(ctx, q.Question, search.Options{Limit: 10, MinScore: 6, SemanticWeight: 10, SemanticFloor: .5, SemanticSlots: 3, Scopes: []string{"benchmark"}})
		must(err)
		rs = benchmarkGates(rs)
		must(enc.Encode(map[string]any{"id": q.ID, "variant": "keyphrases", "results": rs, "latency_ms": float64(time.Since(t).Microseconds()) / 1000}))
		must(f.Close())
		must(db.Close())
		must(os.WriteFile(filepath.Join(path, "done"), []byte("ok\n"), 0600))
		fmt.Fprintf(os.Stderr, "JCODE_PROGRESS {\"current\":%d,\"total\":%d,\"unit\":\"LongMemEval questions\"}\n", n+1, len(qs))
	}
}

func benchmarkGates(rs []search.Result) []search.Result {
	return benchmarkGatesFor(rs, "baseline")
}

func benchmarkGatesFor(rs []search.Result, variant string) []search.Result {
	var kept []search.Result
	for _, r := range rs {
		if variant == "no-semantic-gate" || !r.SemanticOn || r.Semantic >= .64 || r.Coverage >= 1 && r.Semantic >= .58 {
			kept = append(kept, r)
		}
	}
	var out []search.Result
	for _, r := range kept {
		if variant == "no-relative-gate" || r.Total >= .62*kept[0].Total {
			out = append(out, r)
		}
	}
	return out
}

// Command recall4 replays the frozen recall3 real-memory messages through the
// production recall engine with one variant of search options, against a
// scratch copy of the frozen memory snapshot. Scratch only: it refuses any
// database outside $JCODE_SCRATCH_DIR/recall4.
//
//	go run ./muninn/cmd/recall4 -db $JCODE_SCRATCH_DIR/recall4/real/<v>.db \
//	  -input $JCODE_SCRATCH_DIR/recall3/replay.json -variant bm25 > out.jsonl
//
// Tool calls (n7jbmx): a message with "tool" set is one tool call. Variant
// "anchor" runs tool-call recall (recall.ToolRecall) on it; any other
// variant runs message recall on its rendered text. With -asof each call
// sees only memories last written before its timestamp, so a memory written
// from the replayed session itself cannot answer it.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/anchor"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

type message struct {
	ID        int      `json:"id"`
	Home      string   `json:"home"`
	Session   string   `json:"session"`
	Message   string   `json:"message"`
	Cwd       string   `json:"cwd"`
	Split     string   `json:"split"`
	Timestamp string   `json:"timestamp"`
	Tool      string   `json:"tool"`
	Command   string   `json:"command"`
	Paths     []string `json:"paths"`
}

// cachedEmbed caches query vectors on disk so variants share one embedding
// per message (all variants embed the same query text).
type cachedEmbed struct {
	embed.Embedder
	dir string
}

func (c cachedEmbed) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		h := sha256.Sum256([]byte(c.Model() + "\x00" + t))
		p := filepath.Join(c.dir, hex.EncodeToString(h[:]))
		if b, err := os.ReadFile(p); err == nil {
			out[i] = embed.DecodeF32(b)
			continue
		}
		v, err := c.Embedder.Embed(ctx, []string{t})
		if err != nil {
			return nil, err
		}
		out[i] = v[0]
		if err := os.WriteFile(p, embed.EncodeF32(v[0]), 0o600); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// variant adjusts search options and recall gates. Each name is one idea.
// Tool-call parts (anchor, cos60, max40, lim2) set t.
func variant(name string, o *search.Options, e *recall.Engine, t *recall.ToolRequest) {
	for _, part := range strings.Split(name, "+") {
		switch {
		case part == "anchor": // tool-call recall by exact anchors (n7jbmx)
		case part == "shipped": // the arm the hook runs: recall.ToolRequest.Shipped
			*t = t.Shipped()
		case strings.HasPrefix(part, "cos"): // cos60: anchor match and cosine >= 0.60
			var f float64
			fmt.Sscanf(part, "cos%g", &f)
			t.MinCos = f / 100
		case strings.HasPrefix(part, "max"): // max40: an anchor more than 40 memories share never triggers
			fmt.Sscanf(part, "max%d", &t.MaxShared)
		case part == "near": // path anchors end at the path or its folder
			t.Near = true
		case part == "needpath": // a command alone never triggers
			t.NeedPath = true
		case strings.HasPrefix(part, "lim"): // lim2: at most 2 memories per call
			fmt.Sscanf(part, "lim%d", &t.Limit)
		case part == "baseline": // flat points per word (pre-BM25 production)
			o.BM25Scale = 0
		case part == "default": // today's configured defaults
		case part == "bm25": // BM25 at a scale comparable to the old points
			o.BM25Scale = 1
		case strings.HasPrefix(part, "bm25x"): // bm25x2: scale 2
			var f float64
			fmt.Sscanf(part, "bm25x%g", &f)
			o.BM25Scale = f
		case part == "union": // top 20, no gates, no session dedupe: candidate pool for offline arms
			o.Limit, o.MinScore = 20, 0
			e.MinSemantic, e.MinSemanticFull, e.RelativeFloor = 0, 0, 0
		case strings.HasPrefix(part, "norecords"): // norecords600: skip records above 600 bytes
			e.SkipRecordsAbove = 1
			fmt.Sscanf(part, "norecords%d", &e.SkipRecordsAbove)
		case part == "nogates":
			e.MinSemantic, e.MinSemanticFull, e.RelativeFloor = 0, 0, 0
		case part == "norel":
			e.RelativeFloor = 0
		case strings.HasPrefix(part, "rel"): // rel50: relative floor 0.50
			var f float64
			fmt.Sscanf(part, "rel%g", &f)
			e.RelativeFloor = f / 100
		case strings.HasPrefix(part, "min"): // min4: MinScore 4
			fmt.Sscanf(part, "min%g", &o.MinScore)
		default:
			panic("unknown variant part " + part)
		}
	}
}

func main() {
	dbpath := flag.String("db", "", "scratch copy of the frozen snapshot")
	input := flag.String("input", "", "replay JSON")
	name := flag.String("variant", "baseline", "variant, parts joined with +")
	asof := flag.Bool("asof", false, "each message sees only memories last written before its timestamp")
	// Embedder arms (dt02yw, arm.go).
	mode := flag.String("mode", "replay", "replay | reembed | buildarm | latency | qvecs")
	scratch := flag.String("scratch", "recall4", "scratch folder under $JCODE_SCRATCH_DIR that -db must be in")
	armModel := flag.String("embed-model", "", "Ollama embedding model for this arm (default: the configured one, as before)")
	dim := flag.Int("dim", 0, "Matryoshka length: keep the first N dimensions, then L2-renormalise (0 = full)")
	qprefix := flag.String("qprefix", "", "query prefix of the arm")
	dprefix := flag.String("dprefix", "", "document prefix of the arm")
	affine := flag.String("affine", "", "a,b: map the arm's cosine c to a*c+b (nomic scale) so the configured gates apply")
	vectors := flag.String("vectors", "", "buildarm: the store copy reembed wrote the arm's full vectors to")
	maxLoad := flag.Float64("max-load", 20, "reembed: pause while the 1-minute load average is above this (0 = never)")
	flag.Parse()
	root := filepath.Join(os.Getenv("JCODE_SCRATCH_DIR"), *scratch)
	if *scratch == "" || strings.Contains(*scratch, "..") || strings.ContainsRune(*scratch, filepath.Separator) {
		panic("-scratch must be one folder name under $JCODE_SCRATCH_DIR")
	}
	cfg, err := config.Load()
	must(err)
	ar := arm{Model: *armModel, Dim: *dim, QPrefix: *qprefix, DPrefix: *dprefix}
	ar.A, ar.B = parseAffine(*affine)
	if ar.Model != "" {
		// The name is recorded as the embedder model, so the pipeline hash
		// differs per arm. It must not match a PromptsFor family.
		ar.Name = fmt.Sprintf("arm:%s@%d/%s", ar.Model, ar.Dim, *affine)
	}
	if *mode == "latency" || *mode == "qvecs" {
		data, err := os.ReadFile(*input)
		must(err)
		var messages []message
		must(json.Unmarshal(data, &messages))
		if *mode == "latency" {
			writeJSON(latency(messages, ar, cfg.OllamaURL))
		} else {
			writeJSON(qvecs(messages, embed.NewOllama(cfg.OllamaURL, ar.Model, 0), ar.QPrefix))
		}
		return
	}
	abs, _ := filepath.Abs(*dbpath)
	if os.Getenv("JCODE_SCRATCH_DIR") == "" || !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		panic("database must be under $JCODE_SCRATCH_DIR/" + *scratch)
	}
	db, err := store.Open(abs)
	must(err)
	defer db.Close()
	switch *mode {
	case "reembed":
		writeJSON(reembed(db, ar, cfg.OllamaURL, *maxLoad))
		return
	case "buildarm":
		writeJSON(buildarm(db, ar, *vectors))
		return
	case "replay":
	default:
		panic("unknown -mode " + *mode)
	}
	_, err = db.Exec(`DELETE FROM injections`)
	must(err)
	ctx := context.Background()
	qdir := filepath.Join(root, "real", "qvec")
	must(os.MkdirAll(qdir, 0o700))
	var em embed.Embedder = cachedEmbed{embed.NewOllama(cfg.OllamaURL, cfg.EmbedModel, 0), qdir}
	if ar.Model != "" {
		if db.GetMeta("pipeline_hash") != index.PipelineHash(ar.Name) {
			panic("the store holds no vectors of this arm: run -mode buildarm with the same -embed-model, -dim, -affine first")
		}
		em = armEmbed{cachedEmbed{embed.NewOllama(cfg.OllamaURL, ar.Model, 0), qdir}, ar}
	}
	s := &search.Searcher{DB: db, Embedder: em}
	a := &app.App{Cfg: cfg, DB: db, Searcher: s, Embedder: em}
	o := a.RecallOptions()
	e := &recall.Engine{DB: db, Searcher: s, Options: o, RelativeFloor: cfg.RelativeFloor, MinSemantic: cfg.RecallMinSemantic, MinSemanticFull: cfg.RecallMinSemanticFull}
	if anchor.Recipe != db.GetMeta("anchor_recipe") {
		_, err = db.Exec(`BEGIN IMMEDIATE`)
		must(err)
		_, err = index.BackfillAnchors(db)
		must(err)
		_, err = db.Exec(`COMMIT`)
		must(err)
	}
	userHome, err := os.UserHomeDir()
	must(err)
	var tool recall.ToolRequest
	variant(*name, &e.Options, e, &tool)
	anchorArm := strings.Contains("+"+*name+"+", "+anchor+") || strings.Contains("+"+*name+"+", "+shipped+")
	written := map[int64]int64{}
	if *asof {
		rows, err := db.Query(`SELECT id, updated_ns FROM units`)
		must(err)
		for rows.Next() {
			var id, ns int64
			must(rows.Scan(&id, &ns))
			written[id] = ns
		}
		must(rows.Err())
		rows.Close()
	}
	data, err := os.ReadFile(*input)
	must(err)
	var messages []message
	must(json.Unmarshal(data, &messages))
	enc := json.NewEncoder(os.Stdout)
	for _, m := range messages {
		b := a.Bound(m.Home, m.Cwd)
		must(b.Err)
		scopeFor := b.ScopeFor
		if *asof {
			at, err := time.Parse(time.RFC3339Nano, m.Timestamp)
			must(err)
			cut := at.UnixNano()
			scopeFor = func(u store.Unit) (string, bool) {
				if written[u.ID] >= cut {
					return "", false
				}
				return b.ScopeFor(u)
			}
		}
		r := recall.Request{Session: "jcode:" + m.Session, Harness: "jcode", Message: m.Message, Scopes: b.Scopes, ScopeFor: scopeFor}
		if strings.Contains(*name, "union") {
			r.Session = ""
		}
		t := time.Now()
		c, cancel := context.WithTimeout(ctx, 30*time.Second)
		var resp recall.Response
		if anchorArm {
			tr := tool
			tr.Session, tr.Harness, tr.Tool, tr.Command, tr.Touched = r.Session, "jcode", m.Tool, m.Command, m.Paths
			tr.Root, tr.Home, tr.Scopes, tr.ScopeFor = m.Cwd, userHome, b.Scopes, scopeFor
			resp, err = e.ToolRecall(c, tr)
		} else {
			resp, err = e.Recall(c, r)
		}
		cancel()
		errText := ""
		if err != nil {
			errText = err.Error()
		}
		must(enc.Encode(map[string]any{"id": m.ID, "split": m.Split, "response": resp, "latency_ms": float64(time.Since(t).Microseconds()) / 1000, "error": errText, "scopes": b.Scopes, "variant": *name}))
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

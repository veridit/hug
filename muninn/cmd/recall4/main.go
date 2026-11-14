// Command recall4 replays the frozen recall3 real-memory messages through the
// production recall engine with one variant of search options, against a
// scratch copy of the frozen memory snapshot. Scratch only: it refuses any
// database outside $JCODE_SCRATCH_DIR/recall4.
//
//	go run ./muninn/cmd/recall4 -db $JCODE_SCRATCH_DIR/recall4/real/<v>.db \
//	  -input $JCODE_SCRATCH_DIR/recall3/replay.json -variant bm25 > out.jsonl
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

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

type message struct {
	ID        int    `json:"id"`
	Home      string `json:"home"`
	Session   string `json:"session"`
	Message   string `json:"message"`
	Cwd       string `json:"cwd"`
	Split     string `json:"split"`
	Timestamp string `json:"timestamp"`
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
func variant(name string, o *search.Options, e *recall.Engine) {
	for _, part := range strings.Split(name, "+") {
		switch {
		case part == "baseline": // flat points per word (pre-BM25 production)
			o.BM25Scale = 0
		case part == "default": // today's configured defaults
		case part == "bm25": // BM25 at a scale comparable to the old points
			o.BM25Scale = 1
		case strings.HasPrefix(part, "bm25x"): // bm25x2: scale 2
			var f float64
			fmt.Sscanf(part, "bm25x%g", &f)
			o.BM25Scale = f
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
	flag.Parse()
	root := filepath.Join(os.Getenv("JCODE_SCRATCH_DIR"), "recall4")
	abs, _ := filepath.Abs(*dbpath)
	if os.Getenv("JCODE_SCRATCH_DIR") == "" || !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		panic("database must be under $JCODE_SCRATCH_DIR/recall4")
	}
	db, err := store.Open(abs)
	must(err)
	defer db.Close()
	_, err = db.Exec(`DELETE FROM injections`)
	must(err)
	cfg, err := config.Load()
	must(err)
	ctx := context.Background()
	qdir := filepath.Join(root, "real", "qvec")
	must(os.MkdirAll(qdir, 0o700))
	em := cachedEmbed{embed.NewOllama(cfg.OllamaURL, cfg.EmbedModel, 0), qdir}
	s := &search.Searcher{DB: db, Embedder: em}
	a := &app.App{Cfg: cfg, DB: db, Searcher: s, Embedder: em}
	o := a.RecallOptions()
	e := &recall.Engine{DB: db, Searcher: s, Options: o, RelativeFloor: cfg.RelativeFloor, MinSemantic: cfg.RecallMinSemantic, MinSemanticFull: cfg.RecallMinSemanticFull}
	variant(*name, &e.Options, e)
	data, err := os.ReadFile(*input)
	must(err)
	var messages []message
	must(json.Unmarshal(data, &messages))
	enc := json.NewEncoder(os.Stdout)
	for _, m := range messages {
		b := a.Bound(m.Home, m.Cwd)
		must(b.Err)
		r := recall.Request{Session: "jcode:" + m.Session, Harness: "jcode", Message: m.Message, Scopes: b.Scopes, ScopeFor: b.ScopeFor}
		t := time.Now()
		c, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, err := e.Recall(c, r)
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

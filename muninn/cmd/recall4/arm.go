package main

// Embedder arms (dt02yw): replay recall with another Ollama embedding model,
// a Matryoshka length and the model's documented prefixes, on a scratch
// copy whose vectors were built for that arm. Modes:
//
//	-mode reembed   embed every distinct memory text of a store copy with
//	                -embed-model (and -dprefix), timed, into the copy
//	-mode buildarm  write one arm's vectors (truncated to -dim, renormalised,
//	                optionally -affine calibrated) into a snapshot copy
//	-mode latency   time single query embeds on the replay messages
//	-mode qvecs     write the full-length query vectors of the replay messages
//
// Cosine calibration (-affine a,b): the recall gates (semantic floor 0.5,
// min 0.64/0.58, weight 10) were tuned on nomic-embed-text cosines. An arm
// maps its cosine c to a*c+b on nomic's scale, exactly, by appending one
// coordinate: q' = [sqrt(a) q, s sqrt(|b|)], d' = [sqrt(a) d, sqrt(|b|)],
// s = sign(b), so q'.d' = a q.d + b. The sign sketch gains one constant bit,
// which shifts every Hamming distance alike; ranking is unchanged.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/store"
)

// arm describes one embedder arm.
type arm struct {
	Name    string  // recorded as the embedder model, so the pipeline hash differs per arm
	Model   string  // Ollama model
	Dim     int     // Matryoshka length; 0 = full
	QPrefix string  // query prefix
	DPrefix string  // document prefix
	A, B    float64 // affine cosine calibration; A == 0 means none
}

// rawPipeline is the pipeline hash a reembed run stores full vectors under.
func (a arm) rawPipeline() string { return index.PipelineHash("raw:" + a.Model + "\x00" + a.DPrefix) }

// shape truncates to Dim, renormalises and applies the calibration.
func (a arm) shape(v []float32, query bool) []float32 {
	if a.Dim > 0 && a.Dim < len(v) {
		v = append([]float32(nil), v[:a.Dim]...)
		embed.Normalize(v)
	}
	if a.A == 0 {
		return v
	}
	out := make([]float32, len(v)+1)
	sa := float32(math.Sqrt(a.A))
	for i, x := range v {
		out[i] = sa * x
	}
	last := float32(math.Sqrt(math.Abs(a.B)))
	if query && a.B < 0 {
		last = -last
	}
	out[len(v)] = last
	return out
}

// armEmbed is the query side of an arm: prefix, embed (cached), shape.
type armEmbed struct {
	inner embed.Embedder
	a     arm
}

func (e armEmbed) Model() string { return e.a.Name }

func (e armEmbed) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	in := make([]string, len(texts))
	for i, t := range texts {
		in[i] = e.a.QPrefix + t
	}
	vs, err := e.inner.Embed(ctx, in)
	if err != nil {
		return nil, err
	}
	out := make([][]float32, len(vs))
	for i, v := range vs {
		out[i] = e.a.shape(v, true)
	}
	return out, nil
}

func parseAffine(s string) (float64, float64) {
	if s == "" {
		return 0, 0
	}
	p := strings.SplitN(s, ",", 2)
	a, err := strconv.ParseFloat(p[0], 64)
	must(err)
	b, err := strconv.ParseFloat(p[1], 64)
	must(err)
	return a, b
}

// load1 is the 1-minute load average (macOS sysctl, else /proc/loadavg).
func load1() float64 {
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err != nil {
		b, err := os.ReadFile("/proc/loadavg")
		if err != nil {
			return 0
		}
		out = b
	}
	f := strings.Fields(strings.Trim(string(out), "{} \n"))
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(strings.ReplaceAll(f[0], ",", "."), 64)
	return v
}

// waitLoad pauses while the 1-minute load is above max and returns the pause.
func waitLoad(max float64) time.Duration {
	if max <= 0 {
		return 0
	}
	t := time.Now()
	for l := load1(); l > max; l = load1() {
		fmt.Fprintf(os.Stderr, "load %.1f > %.0f, waiting\n", l, max)
		time.Sleep(30 * time.Second)
	}
	return time.Since(t)
}

// reembed embeds every distinct memory text of db under the arm's raw
// pipeline, the work a model switch makes `hug mem index` do. Resumable:
// texts already embedded are skipped and counted.
func reembed(db *store.DB, a arm, url string, maxLoad float64) map[string]any {
	ctx := context.Background()
	pipeline := a.rawPipeline()
	rows, err := db.Query(`SELECT DISTINCT u.content_hash, u.title, u.body FROM units u
	  LEFT JOIN embeddings e ON e.content_hash=u.content_hash AND e.pipeline_hash=?
	  WHERE e.content_hash IS NULL`, pipeline)
	must(err)
	type job struct{ hash, text string }
	var jobs []job
	for rows.Next() {
		var h, t, b string
		must(rows.Scan(&h, &t, &b))
		jobs = append(jobs, job{h, a.DPrefix + index.EmbeddingText(t, b)})
	}
	must(rows.Err())
	rows.Close()
	var total int
	must(db.QueryRow(`SELECT COUNT(DISTINCT content_hash) FROM units`).Scan(&total))
	o := embed.NewOllama(url, a.Model, 0)
	var active, paused time.Duration
	var bytes, dim int
	start := time.Now()
	for s := 0; s < len(jobs); s += embed.BatchSize {
		if (s/embed.BatchSize)%20 == 0 {
			paused += waitLoad(maxLoad)
			fmt.Fprintf(os.Stderr, "JCODE_PROGRESS {\"current\":%d,\"total\":%d,\"unit\":\"texts\"}\n", s, len(jobs))
		}
		e := min(s+embed.BatchSize, len(jobs))
		texts := make([]string, 0, e-s)
		for _, j := range jobs[s:e] {
			texts = append(texts, j.text)
			bytes += len(j.text)
		}
		t := time.Now()
		vecs, err := o.Embed(ctx, texts)
		must(err)
		_, err = db.Exec(`BEGIN IMMEDIATE`)
		must(err)
		for i, v := range vecs {
			dim = len(v)
			must(db.PutEmbedding(jobs[s+i].hash, pipeline, len(v), embed.EncodeF32(v), embed.Quantize(v), embed.Sketch(v)))
		}
		_, err = db.Exec(`COMMIT`)
		must(err)
		active += time.Since(t)
	}
	return map[string]any{"model": a.Model, "dprefix": a.DPrefix, "distinct_texts": total, "embedded": len(jobs),
		"reused": total - len(jobs), "bytes": bytes, "dim": dim, "active_s": active.Seconds(),
		"paused_s": paused.Seconds(), "wall_s": time.Since(start).Seconds(), "texts_per_s": float64(len(jobs)) / max(active.Seconds(), 1e-9)}
}

// buildarm replaces db's vectors with the arm's, read from the reembed copy.
func buildarm(db *store.DB, a arm, vectors string) map[string]any {
	src, err := sql.Open("sqlite", "file:"+vectors+"?mode=ro")
	must(err)
	defer src.Close()
	pipeline := index.PipelineHash(a.Name)
	_, err = db.Exec(`BEGIN IMMEDIATE`)
	must(err)
	for _, t := range []string{"embeddings", "sketches", "qvecs", "neighbors"} {
		_, err = db.Exec(`DELETE FROM ` + t)
		must(err)
	}
	rows, err := db.Query(`SELECT DISTINCT content_hash FROM units`)
	must(err)
	var hashes []string
	for rows.Next() {
		var h string
		must(rows.Scan(&h))
		hashes = append(hashes, h)
	}
	must(rows.Err())
	rows.Close()
	missing, dim := 0, 0
	stmt, err := src.Prepare(`SELECT vec FROM embeddings WHERE content_hash=? AND pipeline_hash=?`)
	must(err)
	for _, h := range hashes {
		var blob []byte
		if err := stmt.QueryRow(h, a.rawPipeline()).Scan(&blob); err == sql.ErrNoRows {
			missing++
			continue
		} else {
			must(err)
		}
		v := a.shape(embed.DecodeF32(blob), false)
		dim = len(v)
		must(db.PutEmbedding(h, pipeline, len(v), embed.EncodeF32(v), embed.Quantize(v), embed.Sketch(v)))
	}
	must(db.SetMeta("pipeline_hash", pipeline))
	must(db.SetMeta("pipeline_query_prefix", ""))
	_, err = db.Exec(`COMMIT`)
	must(err)
	return map[string]any{"arm": a.Name, "hashes": len(hashes), "missing": missing, "stored_dim": dim}
}

// latency times one uncached query embed per message, model warm, in order.
func latency(messages []message, a arm, url string) map[string]any {
	ctx := context.Background()
	o := embed.NewOllama(url, a.Model, 0)
	t := time.Now()
	_, err := o.Embed(ctx, []string{a.QPrefix + "warm up"})
	must(err)
	first := time.Since(t)
	var ms []float64
	var lens []int
	for _, m := range messages {
		q := a.QPrefix + recall.Truncate(strings.TrimSpace(m.Message), recall.MaxQueryBytes)
		t := time.Now()
		_, err := o.Embed(ctx, []string{q})
		must(err)
		ms = append(ms, float64(time.Since(t).Microseconds())/1000)
		lens = append(lens, len(q))
	}
	s := append([]float64(nil), ms...)
	sort.Float64s(s)
	q := func(p float64) float64 { return s[int(math.Min(float64(len(s)-1), math.Floor(p*float64(len(s)))))] }
	over := 0
	for _, x := range ms {
		if x > 200 {
			over++
		}
	}
	return map[string]any{"model": a.Model, "qprefix": a.QPrefix, "n": len(ms), "first_call_ms": float64(first.Microseconds()) / 1000,
		"p50_ms": q(.5), "p95_ms": q(.95), "max_ms": s[len(s)-1], "over_200ms": over, "load1_after": load1(), "ms": ms, "bytes": lens}
}

// qvecs writes the full-length query vectors of the messages (cached).
func qvecs(messages []message, e embed.Embedder, qprefix string) []map[string]any {
	ctx := context.Background()
	var out []map[string]any
	for _, m := range messages {
		msg := strings.TrimSpace(m.Message)
		if msg == "" || recall.IsOwnOutput(msg) || recall.IsHarnessNotice(msg) {
			continue
		}
		v, err := e.Embed(ctx, []string{qprefix + recall.Truncate(msg, recall.MaxQueryBytes)})
		must(err)
		out = append(out, map[string]any{"id": m.ID, "vec": v[0]})
	}
	return out
}

func writeJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	must(enc.Encode(v))
}

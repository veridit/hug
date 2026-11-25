// Package app wires configuration, storage, embedding, indexing and recall.
package app

import (
	"context"
	"fmt"
	"github.com/veridit/hug/internal/env"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
)

// App is an opened muninn instance.
type App struct {
	sourceMu sync.Mutex
	// reindexMu keeps one incremental pass at a time in this process (the MCP
	// server's background startup pass and a write tool's pass).
	reindexMu   sync.Mutex
	sourceRoots map[string]sourceBoundary
	Cfg         config.Config
	DB          *store.DB
	Embedder    embed.Embedder // nil = keyword-only
	EmbedNote   string         // which embedder was chosen and why
	Indexer     *index.Indexer
	Searcher    *search.Searcher
	Recall      *recall.Engine
}

// NativePath is muninn's own memory file.
func NativePath() string { return filepath.Join(config.DataDir(), "memories.json") }

// Open loads config, opens the database and probes Ollama.
func Open(ctx context.Context) (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	db, err := store.Open(config.DBPath())
	if err != nil {
		return nil, err
	}
	a := &App{Cfg: cfg, DB: db}
	a.Embedder, a.EmbedNote = pickEmbedder(ctx, cfg)
	a.Indexer = &index.Indexer{DB: db, Embedder: a.Embedder}
	a.Searcher = &search.Searcher{DB: db, Embedder: a.Embedder}
	minSem, minFull := cfg.RecallMinSemantic, cfg.RecallMinSemanticFull
	if a.Embedder != nil && a.Embedder.Model() == embed.Builtin.Name {
		minSem, minFull = cfg.BuiltinMinSemantic, cfg.BuiltinMinSemanticFull
	}
	a.Recall = &recall.Engine{DB: db, Searcher: a.Searcher, Options: a.RecallOptions(),
		RelativeFloor: cfg.RelativeFloor, MinSemantic: minSem, MinSemanticFull: minFull, SkipRecordsAbove: cfg.SkipRecordsAbove}
	return a, nil
}

// PickEmbedder is pickEmbedder, for a hook that opened the store without
// one and needs it only sometimes (tool-call recall, n7jbmx).
func PickEmbedder(ctx context.Context, cfg config.Config) (embed.Embedder, string) {
	return pickEmbedder(ctx, cfg)
}

// pickEmbedder resolves cfg.Embedder. For the built-in model it prefers a
// running `hug mem serve` (model already loaded) over loading it in-process.
func pickEmbedder(ctx context.Context, cfg config.Config) (embed.Embedder, string) {
	if env.Get(env.NoEmbed) != "" || cfg.Embedder == "none" {
		return nil, "keyword-only (embedder disabled)"
	}
	ollama := func() embed.Embedder {
		o := embed.NewOllama(cfg.OllamaURL, cfg.EmbedModel, time.Duration(cfg.EmbedDelayMS)*time.Millisecond)
		if o.Available(ctx) {
			return o
		}
		return nil
	}
	builtin := func() (embed.Embedder, string) {
		if env.Get(env.Serve) == "" {
			if r := embed.DialRemote(ctx, cfg.ListenAddr, embed.Builtin.Name); r != nil {
				return r, "built-in model via hug mem serve"
			}
		}
		l, err := embed.NewLocal(config.DataDir())
		if err != nil {
			return nil, err.Error()
		}
		return l, "built-in model (in-process)"
	}
	switch cfg.Embedder {
	case "ollama":
		if o := ollama(); o != nil {
			return o, "ollama " + cfg.EmbedModel
		}
		return nil, "keyword-only (Ollama not reachable at " + cfg.OllamaURL + ")"
	case "builtin":
		return builtin()
	default: // auto
		if o := ollama(); o != nil {
			return o, "ollama " + cfg.EmbedModel
		}
		if e, note := builtin(); e != nil {
			return e, note
		}
		return nil, "keyword-only (no Ollama, built-in model not downloaded: run `hug mem model download`)"
	}
}

// Close releases the database.
func (a *App) Close() error { return a.DB.Close() }

// Preview is an App with the config but no database or embedder: enough to
// name a binding and its places (Bound works on it) before Open, which can
// wait on a locked database, returns (hug mcp's initialize, vrx1ja).
func Preview() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return &App{Cfg: cfg}, nil
}

// JcodeInstances lists the jcode installations muninn indexes and hooks.
func (a *App) JcodeInstances() []harness.JcodeInstance {
	home, _ := os.UserHomeDir()
	extra := append([]string(nil), a.Cfg.JcodeHomes...)
	if d := a.Cfg.JcodeMemoryDir; d != "" { // legacy single-dir setting
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			extra = append(extra, filepath.Dir(d))
		}
	}
	if (a.Cfg.JcodeDiscover != nil && !*a.Cfg.JcodeDiscover) || config.Home() != "" {
		// Explicit opt-out, or an isolated HUG_HOME (tests, throwaway
		// homes): never pick up the user's real jcode installations.
		var out []harness.JcodeInstance
		for _, h := range extra {
			if fi, err := os.Stat(h); err == nil && fi.IsDir() {
				out = append(out, harness.InstanceForHome(nil, h))
			}
		}
		return out
	}
	return harness.DiscoverJcode(home, extra)
}

// IndexOptions returns the configured sources.
func (a *App) IndexOptions() index.Options {
	o := index.Options{NativePath: NativePath(), NotesRoots: a.Cfg.NotesRoots}
	for _, i := range a.JcodeInstances() {
		o.Jcode = append(o.Jcode, index.JcodeSource{Dir: filepath.Join(i.Home, "memory"), Prefix: i.Scope()})
	}
	return o
}

// Reindex runs one incremental pass.
// It waits for the write turn (see store.WriteTurn) unless ctx already
// carries it; a NoWait (hook) pass skips instead of queueing. Lock order:
// write turn, then reindexMu, then the native store lock.
func (a *App) Reindex(ctx context.Context) (index.Stats, error) {
	var release func()
	if a.Indexer.NoWait {
		var ok bool
		var err error
		ctx, release, ok, err = a.DB.TryWriteTurn(ctx)
		if err != nil {
			return index.Stats{}, fmt.Errorf("index: %w", err)
		}
		if !ok {
			return index.Stats{}, index.ErrBusy
		}
	} else {
		var err error
		ctx, release, err = a.DB.WriteTurn(ctx)
		if err != nil {
			return index.Stats{}, fmt.Errorf("index: %w", err)
		}
	}
	defer release()
	a.reindexMu.Lock()
	defer a.reindexMu.Unlock()
	st, err := a.Indexer.Run(ctx, a.IndexOptions())
	if err != nil {
		return st, fmt.Errorf("index: %w", err)
	}
	return st, nil
}

// WriteTurn waits for this database's write turn; see store.WriteTurn.
func (a *App) WriteTurn(ctx context.Context) (context.Context, func(), error) {
	return a.DB.WriteTurn(ctx)
}

// RecallOptions are the automatic-recall search settings.
func (a *App) RecallOptions() search.Options {
	o := search.DefaultOptions()
	o.Limit = a.Cfg.MaxResults
	o.MinScore = a.Cfg.MinScore
	o.SemanticWeight = a.Cfg.SemanticWeight
	o.SemanticFloor = a.Cfg.SemanticFloor
	o.SemanticSlots = a.Cfg.SemanticSlots
	o.BM25Scale = a.Cfg.BM25Scale
	return o
}

package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/veridit/hug/internal/env"
	"github.com/veridit/hug/internal/hugconf"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/daemon"
	"github.com/veridit/hug/muninn/internal/embed"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/mcpserver"
	"github.com/veridit/hug/muninn/internal/project"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/tui"
	"github.com/veridit/hug/muninn/internal/web"
)

func ioReadAll(r io.Reader) ([]byte, error) { return io.ReadAll(io.LimitReader(r, 4<<20)) }

func serveCmd() *cobra.Command {
	var addr string
	var noWatch bool
	var view viewFlags
	c := &cobra.Command{Use: "serve", Short: "Run the watcher (incremental reindex on change) and the local web UI",
		RunE: func(*cobra.Command, []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			// serve is the process that holds the built-in model for hooks;
			// it must load it itself, never ask another serve.
			os.Setenv(env.Serve, "1")
			return withApp(func(_ context.Context, a *app.App) error {
				if addr == "" {
					addr = a.Cfg.ListenAddr
				}
				fmt.Println("embedder:", a.EmbedNote)
				mu := &sync.Mutex{}
				if !noWatch {
					go func() {
						if err := daemon.Watch(ctx, a, mu, daemon.StdLog); err != nil {
							daemon.StdLog("watcher stopped: %v", err)
						}
					}()
				}
				fmt.Printf("hug memory web UI on http://%s\n", addr)
				return web.ListenAndServe(ctx, addr, (&web.Server{App: a, Mu: mu, Binding: view.binding(a)}).Handler())
			})
		}}
	c.Flags().StringVar(&addr, "addr", "", "loopback listen address (default from config, 127.0.0.1:7431)")
	c.Flags().BoolVar(&noWatch, "no-watch", false, "do not watch sources")
	view.flags(c)
	return c
}

func mcpCmd() *cobra.Command {
	var jcodeHome string
	c := &cobra.Command{Use: "mcp", Short: "Serve MCP over stdio (memory_search, memory_remember, ...)",
		RunE: func(*cobra.Command, []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return withApp(func(_ context.Context, a *app.App) error {
				// 2026-11-10: the Claude app gave up on muninn after 2 minutes without an
				// initialize reply, because startup reindex waited on another process's
				// write lock. Serve at once; tools read the index as it stands, and the
				// background pass brings it up to date.
				go func() {
					if _, err := a.Reindex(ctx); err != nil && ctx.Err() == nil {
						fmt.Fprintln(os.Stderr, "hug mem: reindex:", err)
					}
				}()
				cwd, _ := os.Getwd()
				return mcpserver.RunStdio(ctx, a, Version, a.Bound(jcodeHome, cwd))
			})
		}}
	c.Flags().StringVar(&jcodeHome, "jcode-home", "", "serve one jcode instance: same scopes as its recall hook (default $JCODE_HOME when set)")
	return c
}

func tuiCmd() *cobra.Command {
	var view viewFlags
	c := &cobra.Command{Use: "tui", Short: "Interactive terminal browser", RunE: func(*cobra.Command, []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			if _, err := a.Reindex(ctx); err != nil {
				return err
			}
			return tui.Run(a, view.binding(a))
		})
	}}
	view.flags(c)
	return c
}

// selfPath is the hug hooks and MCP configs should run (ynh84t): the hug
// next to this program as invoked, so a Homebrew link stays a link and
// never becomes a versioned Cellar path. Hooks run `hug mem hook X`.
func selfPath() string { return hugconf.HugPath() }

var installers = map[string]func(bin string, install, dry bool) (harness.InstallResult, error){
	"claude": harness.InstallClaude,
	"codex":  harness.InstallCodex,
	"jcode":  harness.InstallJcode,
}

func installLike(install bool) *cobra.Command {
	var dry, allInstances bool
	var bin string
	var homes []string
	use, short := "install <claude|codex|jcode|all>", "Wire automatic recall into a harness (keeps a backup)"
	if !install {
		use, short = "uninstall <claude|codex|jcode|all>", "Remove hug's memory hook from a harness"
	}
	c := &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		if bin == "" {
			bin = selfPath()
		}
		names := []string{args[0]}
		if args[0] == "all" {
			names = []string{"claude", "codex", "jcode"}
		}
		for _, n := range names {
			if n == "jcode" && (allInstances || len(homes) > 0) {
				if err := installJcodeInstances(bin, homes, allInstances, install, dry); err != nil {
					return err
				}
				continue
			}
			f, ok := installers[n]
			if !ok {
				return fmt.Errorf("unknown harness %q", n)
			}
			r, err := f(bin, install, dry)
			if err != nil {
				return fmt.Errorf("%s: %w", n, err)
			}
			if dry {
				fmt.Print("[dry-run] ")
			}
			fmt.Println(r)
		}
		// One command should leave a working setup: fetch the built-in
		// embedding model (pinned, verified) unless Ollama already serves one.
		if install && !dry && !embed.Builtin.Present(config.DataDir()) {
			cfg, _ := config.Load()
			o := embed.NewOllama(cfg.OllamaURL, cfg.EmbedModel, 0)
			if cfg.Embedder == "builtin" || (cfg.Embedder == "auto" && !o.Available(context.Background())) {
				fmt.Fprintln(os.Stderr, "fetching the built-in embedding model (~127 MB, once; pinned and sha256-verified)")
				if err := embed.Builtin.Download(context.Background(), config.DataDir(), nil); err != nil {
					fmt.Fprintln(os.Stderr, "model download failed (muninn works keyword-only until `hug mem model download` succeeds):", err)
				}
			}
		}
		return nil
	}}
	c.Flags().BoolVar(&dry, "dry-run", false, "show what would change")
	c.Flags().StringVar(&bin, "bin", "", "hug binary path to put in the hook (default: this hug)")
	c.Flags().BoolVar(&allInstances, "all-instances", false, "jcode: every instance (~/.jcode, ~/.zshrc JCODE_INSTANCES, config jcode_homes)")
	c.Flags().StringSliceVar(&homes, "jcode-home", nil, "jcode: a specific JCODE_HOME (repeatable)")
	return c
}

func installJcodeInstances(bin string, homes []string, all, install, dry bool) error {
	var targets []harness.JcodeInstance
	if all {
		if err := withApp(func(_ context.Context, a *app.App) error {
			targets = a.JcodeInstances()
			return nil
		}); err != nil {
			return err
		}
	}
	for _, h := range homes {
		targets = append(targets, harness.InstanceForHome(targets, h))
	}
	seen := map[string]bool{}
	for _, t := range targets {
		if seen[t.Home] {
			continue
		}
		seen[t.Home] = true
		r, err := harness.InstallJcodeHome(t.Home, "jcode["+t.Name+"]", bin, install, dry)
		if err != nil {
			return fmt.Errorf("jcode[%s]: %w", t.Name, err)
		}
		if dry {
			fmt.Print("[dry-run] ")
		}
		fmt.Println(r)
	}
	return nil
}

func jcodeCmd() *cobra.Command {
	c := &cobra.Command{Use: "jcode", Short: "jcode integration helpers"}
	c.AddCommand(&cobra.Command{Use: "instances", Short: "List jcode instances hug mem knows about", RunE: func(*cobra.Command, []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			counts := map[string]int{}
			rows, err := a.DB.QueryContext(ctx, `SELECT scope, COUNT(*) FROM units WHERE active=1 AND scope LIKE 'jcode@%' GROUP BY scope`)
			if err != nil {
				return err
			}
			for rows.Next() {
				var s string
				var n int
				rows.Scan(&s, &n)
				if i := strings.Index(s, ":"); i > 0 {
					s = s[:i]
				}
				counts[s] += n
			}
			rows.Close()
			fmt.Printf("%-26s %-8s %-6s %-9s %s\n", "INSTANCE", "MEMORIES", "BINARY", "HOOK", "JCODE_HOME")
			for _, i := range a.JcodeInstances() {
				hook := "no"
				if harness.JcodeHookInstalled(i.Home) {
					hook = "yes"
				}
				bin := "PATH"
				if i.Binary() != "" {
					bin = "own"
				}
				fmt.Printf("%-26s %-8d %-6s %-9s %s\n", i.Name, counts[i.Scope()], bin, hook, i.Home)
			}
			return nil
		})
	}})
	return c
}

func installCmd() *cobra.Command   { return installLike(true) }
func uninstallCmd() *cobra.Command { return installLike(false) }

// OpenMCP opens memory bound as "muninn mcp --jcode-home jcodeHome" would be,
// starts the background reindex, and returns the server instructions plus a
// function that adds the memory_* tools to a server. close releases the store.
//
// It returns at once (vrx1ja): opening the store can wait minutes on a
// locked database (store.Open's schema waits busy_timeout under the init
// lock, and several openers queue), and an app gives up on initialize after
// 60 to 120 s. The store, embedder and binding open in the background; the
// instructions come from a binding resolved without the database, and a
// memory tool called before memory is ready answers that memory is still
// opening, with a fix: line. The tools have the same names and schemas
// either way.
func OpenMCP(ctx context.Context, jcodeHome string) (instructions string, register func(*mcp.Server), closeFn func(), err error) {
	pre, err := app.Preview()
	if err != nil {
		return "", nil, nil, err
	}
	cwd, _ := os.Getwd()
	pb := pre.Bound(jcodeHome, cwd)
	if pb.Err != nil {
		return "", nil, nil, pb.Err
	}
	m := openMemory(ctx, jcodeHome, cwd)
	register = func(s *mcp.Server) { mcpserver.RegisterLazy(s, m.get) }
	return mcpserver.InstructionsFor(pb), register, m.close, nil
}

// mcpMemory is memory opening in the background for hug mcp.
type mcpMemory struct {
	ready   chan struct{} // closed when the open finished, either way
	started time.Time

	mu     sync.Mutex
	a      *app.App
	b      app.Binding
	err    error
	closed bool
}

// openRetry is the pause before memory opens again after a busy database.
var openRetry = 5 * time.Second

// busy says the store could not open only because another process held the
// database (SQLITE_BUSY after busy_timeout), or held the schema update while
// this hug had schema to add (store.ErrBusy, 6gtv7a): it opens later.
func busy(err error) bool {
	if errors.Is(err, store.ErrBusy) {
		return true
	}
	s := err.Error()
	return strings.Contains(s, "SQLITE_BUSY") || strings.Contains(s, "database is locked")
}

func openMemory(ctx context.Context, jcodeHome, cwd string) *mcpMemory {
	m := &mcpMemory{ready: make(chan struct{}), started: time.Now()}
	go func() {
		defer close(m.ready)
		a, err := app.Open(ctx)
		// A busy database is another hug writing (an index pass can hold it
		// for minutes): try again, the tools say memory is still opening.
		for err != nil && busy(err) && ctx.Err() == nil {
			fmt.Fprintln(os.Stderr, "hug mcp: memory busy, opening again in", openRetry, ":", err)
			select {
			case <-ctx.Done():
			case <-time.After(openRetry):
			}
			m.mu.Lock()
			closed := m.closed
			m.mu.Unlock()
			if closed || ctx.Err() != nil {
				break
			}
			a, err = app.Open(ctx)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "hug mcp: memory unavailable:", err)
			m.mu.Lock()
			m.err = err
			m.mu.Unlock()
			return
		}
		b := a.Bound(jcodeHome, cwd)
		if b.Err != nil {
			fmt.Fprintln(os.Stderr, "hug mcp: memory unavailable:", b.Err)
			a.Close()
			m.mu.Lock()
			m.err = b.Err
			m.mu.Unlock()
			return
		}
		b = mcpserver.Bind(a, b)
		// Project memories reach the other clones in the background (0hkaa5).
		// Set before the tools can see memory: a tool reads OnWrite.
		project.SpawnSelf()
		mcpserver.OnWrite = func(scope string) {
			if scope == "" || shared(scope) {
				project.Nudge(cwd)
			}
		}
		m.mu.Lock()
		if m.closed {
			m.mu.Unlock()
			a.Close()
			return
		}
		m.a, m.b = a, b
		m.mu.Unlock()
		go func() {
			if _, err := a.Reindex(ctx); err != nil && ctx.Err() == nil {
				fmt.Fprintln(os.Stderr, "hug mem: reindex:", err)
			}
		}()
	}()
	return m
}

// memoryWait is how long a memory tool waits for memory still opening
// before it answers that memory is opening: a normal open takes well under a
// second (0.1-0.3 s measured on a 900 MB store), a blocked one minutes.
const memoryWait = 5 * time.Second

// get is the memory tools' Opener: the open memory, or an error that says
// it is still opening (or why it failed), each with a fix: line.
func (m *mcpMemory) get(ctx context.Context) (*app.App, app.Binding, error) {
	t := time.NewTimer(memoryWait)
	defer t.Stop()
	select {
	case <-m.ready:
	case <-ctx.Done():
		return nil, app.Binding{}, ctx.Err()
	case <-t.C:
		waited := time.Since(m.started).Round(time.Second)
		return nil, app.Binding{}, fmt.Errorf("hug mcp: memory is still opening (%s so far; the memory database may be busy with another hug); the task_* tools work meanwhile\nfix: call this tool again in a minute; hug mem stats in a shell shows the store", waited)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, app.Binding{}, fmt.Errorf("hug mcp: memory is unavailable in this session: %w\nfix: run hug mem stats in a shell to see why, then restart this MCP server", m.err)
	}
	if m.closed {
		return nil, app.Binding{}, fmt.Errorf("hug mcp: this server is closing\nfix: restart this MCP server")
	}
	return m.a, m.b, nil
}

func (m *mcpMemory) close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	if m.a != nil {
		m.a.Close()
		m.a = nil
	}
}

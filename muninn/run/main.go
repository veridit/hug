package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	huginn "github.com/veridit/hug/huginn/run"
	"github.com/veridit/hug/internal/env"
	"io"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/veridit/hug/internal/throttle"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/project"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/trigger"
)

type viewFlags struct {
	all    bool
	scopes []string
	home   string
}

func (v *viewFlags) flags(c *cobra.Command) {
	c.Flags().BoolVar(&v.all, "all", false, "owner view: read all instances (writes remain bound)")
	c.Flags().StringSliceVar(&v.scopes, "scope", nil, scopeHelp)
	c.Flags().StringVar(&v.home, "jcode-home", "", "bind to this JCODE_HOME (default $JCODE_HOME, otherwise global)")
	c.MarkFlagsMutuallyExclusive("all", "scope")
}
func (v viewFlags) binding(a *app.App) app.Binding {
	cwd, _ := os.Getwd()
	return a.Bound(v.home, cwd).View(v.all, v.scopes)
}

// Version is set by the release build (-X).
var Version = "dev"

// ExitNotStored is the exit code of hug mem remember when it refused to
// store (a near-duplicate, a possible secret, a project-specific note sent
// global): the output starts "NOT STORED:" and asks its question (p29rbz).
// Other errors exit 1.
const ExitNotStored = 3

// Main runs muninn with args (without the program name) and returns the exit code.
func Main(args []string) int {
	root := &cobra.Command{
		Use:           "muninn",
		Annotations:   map[string]string{cobra.CommandDisplayNameAnnotation: "hug mem"},
		Short:         "Local-first memory recall for AI coding agents",
		Example:       "  hug mem search \"banana mode\"\n  hug mem forget k7m2qx\n  hug mem remember \"The corrected fact\"",
		SilenceUsage:  true,
		SilenceErrors: true,
		// No completion command: its scripts complete the command name
		// muninn (cobra takes it from Root().Name()), and users type hug
		// (p277bp, crab 22:21). The hidden __complete command stays, so
		// scripts installed before keep answering.
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		// Background work must not spike the machine: it may take longer.
		// The recall hook and interactive commands keep full speed because a
		// user is waiting on them.
		PersistentPreRun: func(c *cobra.Command, _ []string) {
			if backgroundCommands[topLevel(c)] {
				throttle.Background()
			}
		},
		// Every interactive command ends with the likely next step, like
		// hug task does. Hooks, servers and JSON output stay machine-clean.
		PersistentPostRun: func(c *cobra.Command, _ []string) {
			if hint := nextHint(c); hint != "" {
				// stdout stays parseable for commands that print JSON
				// (stats, config): the hint goes to stderr there.
				w := c.OutOrStdout()
				if jsonOutput[topLevel(c)] {
					w = c.ErrOrStderr()
				}
				fmt.Fprintln(w, "next: "+hint)
			}
		},
	}
	root.AddCommand(versionCmd(), dreamCmd(), typesCmd(), indexCmd(), searchCmd(), recallCmd(), rememberCmd(), forgetCmd(), showCmd(), rescopeCmd(),
		listCmd(), relatedCmd(), statsCmd(), gcCmd(), hookCmd(), serveCmd(), mcpCmd(), tuiCmd(),
		installCmd(), uninstallCmd(), configCmd(), jcodeCmd(), importCmd(), projectCmd(), syncCmd(), modelCmd(), captureCmd(), tidyCmd(), upgradeCmd())
	if p := env.Get(env.CPUProfile); p != "" {
		if f, err := os.Create(p); err == nil {
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}
	}
	root.SetArgs(args)
	project.SpawnSelf()
	// Writes queue for the write turn; a person waiting sees why.
	store.TurnNotice = func(holder string, waited time.Duration) {
		fmt.Fprintf(os.Stderr, "hug mem: waiting %s for the write turn, held by %s\n", waited.Round(time.Second), holder)
	}
	if err := root.Execute(); err != nil {
		pprof.StopCPUProfile()
		if _, ok := err.(pendingErr); ok {
			return ExitUpgradePending
		}
		// A refusal of remember is printed as is, first, with its own
		// exit code: it must never read as success (p29rbz).
		if app.Refusal(err) {
			fmt.Fprintln(os.Stderr, err)
			return ExitNotStored
		}
		err = store.ExplainBusy(config.DBPath(), err)
		fmt.Fprintln(os.Stderr, "hug mem:", err)
		if strings.HasPrefix(err.Error(), "unknown command") {
			fmt.Fprintln(os.Stderr, "fix: hug mem --help")
		}
		return 1
	}
	return 0
}

// backgroundCommands run at lowered priority with bounded parallelism.
var backgroundCommands = map[string]bool{
	"capture": true, "index": true, "serve": true, "mcp": true,
	"gc": true, "import": true, "sync": true, "tidy": true, "dream": true,
}

func topLevel(c *cobra.Command) string {
	for c.HasParent() && c.Parent().HasParent() {
		c = c.Parent()
	}
	return c.Name()
}

// openApp opens the app for a command; tests swap it to give commands a
// fixture's app (with its synthetic embedder).
var openApp = app.Open

func withApp(fn func(ctx context.Context, a *app.App) error) error {
	ctx := context.Background()
	a, err := openApp(ctx)
	if err != nil {
		return err
	}
	defer a.Close()
	cwd, _ := os.Getwd()
	if b := a.Bound("", cwd); b.Err != nil {
		return b.Err
	}
	return fn(ctx, a)
}

func versionCmd() *cobra.Command {
	return &cobra.Command{Use: "version", Short: "Print version", Run: func(*cobra.Command, []string) {
		fmt.Println("hug", Version)
	}}
}

func indexCmd() *cobra.Command {
	return &cobra.Command{Use: "index", Short: "Incrementally index all configured sources", RunE: func(*cobra.Command, []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			fmt.Fprintln(os.Stderr, "embedder:", a.EmbedNote)
			st, err := a.Reindex(ctx)
			if err != nil {
				return err
			}
			if err := a.RebuildLegacyFolders(ctx); err != nil {
				return fmt.Errorf("legacy folder cache: %w", err)
			}
			fmt.Println(st)
			if a.Indexer.GraphMoved != "" {
				fmt.Println(a.Indexer.GraphMoved)
			}
			if st.EmbedErr != nil {
				fmt.Fprintln(os.Stderr, "embedding skipped:", st.EmbedErr)
			}
			return nil
		})
	}}
}

func printResults(res []search.Result, explain bool) {
	if len(res) == 0 {
		fmt.Println("no results")
		return
	}
	for _, r := range res {
		body := strings.Join(strings.Fields(r.Unit.Body), " ")
		fmt.Printf("[%s] %s (%s, %s) score %.2f\n", r.Unit.SID, recall.Truncate(body, 300), r.Unit.Kind, r.Unit.Scope, r.Total)
		if explain {
			fmt.Println("    ", r.Explain(), "origin:", r.Unit.Origin)
		}
	}
}

const scopeHelp = "only these scopes (repeatable; trailing * is a prefix match, e.g. 'jcode@seid:*')"

func searchCmd() *cobra.Command {
	var limit int
	var explain, asJSON bool
	var minScore float64
	var view viewFlags
	c := &cobra.Command{Use: "search <query>", Short: "Search memories (keyword + community + semantic)", Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				// Other clones' project memories arrive in the background (0hkaa5).
				cwd, _ := os.Getwd()
				project.NudgeIfStale(cwd)
				o := a.RecallOptions()
				o.Limit, o.MinScore = limit, minScore
				b := view.binding(a)
				o.Scopes, o.ScopeFor = b.Scopes, b.ScopeFor
				res, err := a.Searcher.Search(ctx, strings.Join(args, " "), o)
				if err != nil {
					return err
				}
				if asJSON {
					out := memoriesJSON{Memories: []memoryJSON{}}
					for _, r := range res {
						m := unitJSON(r.Unit)
						score := r.Total
						m.Score = &score
						out.Memories = append(out.Memories, m)
					}
					return json.NewEncoder(os.Stdout).Encode(out)
				}
				printResults(res, explain)
				return nil
			})
		}}
	c.Flags().IntVarP(&limit, "limit", "n", 10, "max results")
	c.Flags().BoolVar(&explain, "explain", false, "show per-signal scores")
	c.Flags().BoolVar(&asJSON, "json", false, "JSON output: {\"memories\": [...]}, each with id, kind, scope, category, type, tags, text, updated, score")
	c.Flags().Float64Var(&minScore, "min-score", 0, "drop results below this total")
	view.flags(c)
	return c
}

func recallCmd() *cobra.Command {
	var session, harnessName string
	var stdin bool
	var view viewFlags
	c := &cobra.Command{Use: "recall [message]", Short: "Automatic-recall block for a message (generic harness entry point)",
		RunE: func(_ *cobra.Command, args []string) error {
			msg := strings.Join(args, " ")
			if stdin {
				b, _ := ioReadAll(os.Stdin)
				msg = string(b)
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				// Bring in other clones' project memories when the last pass
				// is stale; never waits (0hkaa5).
				cwd, _ := os.Getwd()
				project.NudgeIfStale(cwd)
				if _, err := ensureFresh(ctx, a); err != nil {
					return err
				}
				binding := view.binding(a)
				resp, err := a.Recall.Recall(ctx, recall.Request{Session: session, Harness: harnessName, Message: msg, Scopes: binding.Scopes, ScopeFor: binding.ScopeFor})
				if err != nil {
					return err
				}
				if resp.Block != "" {
					fmt.Println(resp.Block)
				}
				return nil
			})
		}}
	c.Flags().StringVar(&session, "session", "", "session id for per-session dedupe")
	c.Flags().StringVar(&harnessName, "harness", "cli", "harness name for the injection log")
	c.Flags().BoolVar(&stdin, "stdin", false, "read the message from stdin")
	view.flags(c)
	return c
}

// ensureFresh runs a cheap incremental pass (stat-only when nothing changed).
func ensureFresh(ctx context.Context, a *app.App) (index.Stats, error) {
	// Recall paths must stay fast: never embed inline beyond a handful of
	// brand-new memories (one Ollama batch is ~0.2-2 s). Larger backlogs,
	// including a model or prompt change, are left to `hug mem index`/`serve`.
	a.Indexer.MaxEmbed = 8
	// If another process is indexing, do not queue behind it: WAL still lets
	// this process read a consistent snapshot, so recall just runs on that.
	a.Indexer.NoWait = true
	defer func() { a.Indexer.MaxEmbed, a.Indexer.NoWait = 0, false }()
	st, err := a.Reindex(ctx)
	if errors.Is(err, index.ErrBusy) {
		return st, nil
	}
	return st, err
}

func rememberCmd() *cobra.Command {
	var category, scope, home, safeReason, place, placeReason, newReason, reinforce, typ, holdsCond, supersedes, reason string
	var when, whenNot string
	var casePaths, caseTools, caseScopes []string
	var noTrigger bool
	var tags []string
	var inProject bool
	c := &cobra.Command{Use: "remember <text>", Short: "Store a memory", Args: cobra.MinimumNArgs(1),
		Long: "Store a memory. The text is the note shown when it is recalled.\n\nFor a rule or pattern, say when it applies (jrj1vq): --when names the situation where an agent needs the note, as an activity (\"reviewing a ticket's history\", not \"tickets\"); --when-not names look-alike situations where it does not help; --case-scope, --case-path and --case-tool bind it to projects, files and tools (each repeatable). The text is then the THEN. A plain memory without them stays valid. A changed trigger on an existing note is a correction: --supersedes ID with the new --when, or --supersedes ID --no-trigger to clear it.\n\nWhen remember refuses (a near-duplicate, a possible secret, a project-specific note sent global, or a changed trigger) it stores nothing: the output starts \"NOT STORED:\", asks its question with the flag that answers it, and the exit code is 3. Other errors exit 1.",
		RunE: func(_ *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				cwd, _ := os.Getwd()
				b := a.Bound(home, cwd)
				if inProject && scope == "" {
					// A client root is that client's global, not a separate project.
					if places := b.Places(); b.Client != "" && len(places) == 1 {
						scope = places[0].Scope
					} else {
						p, err := project.Find(cwd, true)
						if err != nil {
							return err
						}
						scope = p.Scope()
						b = b.Refresh(a)
					}
				}
				ctx = app.WithFix(ctx, a.HarnessFix("cli", reason))
				u, reinforced, err := b.RememberPlaced(ctx, a, strings.Join(args, " "), category, scope, tags, safeReason, place, placeReason, app.RememberDecision{NewReason: newReason, Reinforce: reinforce, Type: typ, Holds: holdsCond, Supersedes: supersedes, Trigger: cliTrigger(when, whenNot, caseScopes, casePaths, caseTools), ClearTrigger: noTrigger})
				if err != nil {
					return err
				}
				// A project memory goes to the other clones in the background (0hkaa5).
				if shared(u.Scope) {
					project.Nudge(cwd)
				}
				if reinforced {
					fmt.Printf("reinforced %s (place: %s)\n", u.SID, b.PlaceForScope(u.Scope).Name)
				} else {
					fmt.Printf("remembered %s (place: %s)\n", u.SID, b.PlaceForScope(u.Scope).Name)
				}
				for _, p := range b.Places() {
					fmt.Printf("available: %s (%s)\n", p.Name, p.Meaning)
				}
				return nil
			})
		}}
	c.Flags().StringVar(&category, "category", "fact", "fact | preference | correction | entity")
	c.Flags().StringVar(&typ, "type", "", "rule (an instruction) | preference | fact (durable knowledge) | record (something that happened)")
	c.Flags().StringVar(&holdsCond, "holds", "", "how long it is true, when you know: \"until 2026-12-01\", \"while task:ID open\", \"while branch NAME unmerged\"; when it ends, recall stops showing it and a dream item asks to retire or keep it")
	c.Flags().StringVar(&when, "when", "", "the situation where an agent needs this note, as an activity (\"choosing a model for a coding subagent\"); the text is then what is shown")
	c.Flags().StringVar(&whenNot, "when-not", "", "look-alike situations where the note does not help (same ticket or tool, another activity)")
	c.Flags().StringArrayVar(&caseScopes, "case-scope", nil, "a project or scope the note is bound to (repeatable)")
	c.Flags().StringArrayVar(&casePaths, "case-path", nil, "a file path or glob the note is bound to (repeatable)")
	c.Flags().StringArrayVar(&caseTools, "case-tool", nil, "a tool, command or service the note is bound to (repeatable)")
	c.Flags().BoolVar(&noTrigger, "no-trigger", false, "with --supersedes ID and the same text: clear that memory's --when/--when-not/--case-*")
	c.Flags().StringVar(&scope, "scope", "", "literal memory scope (overrides --place)")
	c.Flags().StringVar(&place, "place", "", "here (default), client, or global; when unsure choose here")
	c.Flags().StringVar(&placeReason, "place-reason", "", "why this applies beyond this project, when the project-mention check asks")
	c.Flags().StringVar(&newReason, "new-reason", "", "why a similar memory is a different fact, when the duplicate check asks")
	c.Flags().StringVar(&reinforce, "reinforce", "", "reinforce this existing memory instead of adding, answering the duplicate question")
	c.Flags().StringVar(&supersedes, "supersedes", "", "the id of the memory this one corrects; recorded in the fix trail (hug mem stats --fixes), it forgets nothing: hug mem forget ID if the old one no longer holds")
	c.Flags().StringVar(&reason, "reason", "", "with --reinforce or --supersedes: why, kept in the fix trail")
	c.Flags().StringVar(&home, "jcode-home", "", "bind to this JCODE_HOME")
	c.Flags().StringSliceVar(&tags, "tag", nil, "tag (repeatable)")
	c.Flags().StringVar(&safeReason, "safe-reason", "", "why text that looks secret is safe to store (asked for when the secret check fires)")
	c.Flags().BoolVar(&inProject, "project", false, "store in this repository's project scope (shared with the project's other clones in the background, or now with `hug mem sync`)")
	return c
}

func forgetCmd() *cobra.Command {
	var view viewFlags
	var stdin, asJSON bool
	var reason string
	c := &cobra.Command{Use: "forget <memory-id>... | --stdin", Short: "Retire memories or suppress upstream memories locally (several ids: one transaction)",
		Example: "  hug mem forget k7m2qx\n  hug mem forget k7m2qx 3nfhgr 9xybgp\n  hug mem list --json | jq -r '.memories[].id' | hug mem forget --stdin",
		RunE: func(_ *cobra.Command, args []string) error {
			ids := args
			if stdin || (len(args) == 1 && args[0] == "-") {
				if stdin && len(args) > 0 {
					return fmt.Errorf("give ids as arguments or on stdin, not both\nfix: hug mem forget --stdin < ids.txt")
				}
				b, err := ioReadAll(os.Stdin)
				if err != nil {
					return err
				}
				ids = forgetIDs(string(b))
			}
			if len(ids) == 0 {
				return fmt.Errorf("no memory id given\nfix: hug mem forget ID [ID...], or one id per line: hug mem forget --stdin")
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				// One write turn for the whole call: the freshness check, the
				// checks and the forget queue once, not once per step. A full
				// pass only when the index is not current (4vd1mz).
				ctx, release, err := a.WriteTurn(ctx)
				if err != nil {
					return err
				}
				defer release()
				if err := a.FreshFor(ctx, ids); err != nil {
					return err
				}
				// The fix trail (rjk44w): who forgot, and why when given.
				ctx = app.WithFix(ctx, a.HarnessFix("cli", reason))
				out, err := view.binding(a).ForgetMany(ctx, a, ids, app.HarnessSessions()...)
				if err != nil {
					return err
				}
				// The forget reaches the other clones in the background (0hkaa5).
				cwd, _ := os.Getwd()
				project.Nudge(cwd)
				if asJSON {
					return json.NewEncoder(os.Stdout).Encode(struct {
						Forgotten []app.ForgetResult `json:"forgotten"`
					}{out})
				}
				for _, r := range out {
					fmt.Println(r.Message)
				}
				return nil
			})
		}}
	c.Flags().BoolVar(&stdin, "stdin", false, "read memory ids from stdin, one per line (first field; # starts a comment)")
	c.Flags().BoolVar(&asJSON, "json", false, "JSON output: {\"forgotten\": [...]}")
	c.Flags().StringVar(&reason, "reason", "", "why this memory no longer holds (kept in the fix trail: hug mem stats --fixes)")
	view.flags(c)
	return c
}

// forgetIDs reads one id per line: the first field counts, # starts a
// comment, so `hug mem list` lines ("[k7m2qx] (...)") or TSV with the id
// first work as they are. Repeats are dropped.
func forgetIDs(s string) []string {
	var ids []string
	seen := map[string]bool{}
	for _, line := range strings.Split(s, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		id := strings.Trim(f[0], "[],\"")
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

func rescopeCmd() *cobra.Command {
	var target, from string
	var cross, dry bool
	c := &cobra.Command{Use: "rescope <memory-id> --to SCOPE | --from FILE", Short: "Admin: move owned memories to another scope (one, or a batch from a file)",
		Example: "  hug mem rescope k7m2qx --to project:rc-0b343f3c990286d6\n  hug mem rescope --from moves.tsv --dry-run   # lines: <memory-id> <scope>, # comments\n  cut -f1,3 plan.tsv | hug mem rescope --from -",
		Args: func(_ *cobra.Command, args []string) error {
			if from != "" {
				if len(args) != 0 || target != "" {
					return fmt.Errorf("--from takes no memory id or --to; put both on each line: <memory-id> <scope>")
				}
				return nil
			}
			if len(args) != 1 || target == "" {
				return fmt.Errorf("give <memory-id> --to SCOPE, or --from FILE for a batch")
			}
			return nil
		}, RunE: func(_ *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				cwd, _ := os.Getwd()
				// Who moved what lands in the fix trail (j13xs0).
				ctx = app.WithFix(ctx, a.HarnessFix("cli", ""))
				if from != "" {
					items, err := readRescopeItems(from)
					if err != nil {
						return err
					}
					res, err := a.Bound("", cwd).RescopeBatch(ctx, a, items, cross, dry)
					if err != nil {
						return err
					}
					counts := map[string]int{}
					for _, m := range res.Moved {
						counts[m.Target]++
					}
					verb := "rescoped"
					if dry {
						verb = "would rescope"
					}
					for t, n := range counts {
						fmt.Printf("%s %d to %s\n", verb, n, t)
					}
					fmt.Printf("%s %d memories\n", verb, len(res.Moved))
					if res.Backup != "" {
						fmt.Printf("backup: %s (each file next to a .path naming where it goes back)\n", res.Backup)
					}
					return nil
				}
				id, err := a.Bound("", cwd).Rescope(ctx, a, args[0], target, cross)
				if err != nil {
					return err
				}
				fmt.Printf("rescoped %s to %s\n", id, target)
				return nil
			})
		}}
	c.Flags().StringVar(&target, "to", "", "literal target scope")
	c.Flags().BoolVar(&cross, "cross-client", false, "admin approval to cross a client boundary")
	c.Flags().StringVar(&from, "from", "", "batch: file of '<memory-id> <scope>' lines ('-' for stdin); all checked first, one write, one reindex")
	c.Flags().BoolVar(&dry, "dry-run", false, "with --from: check every line and print counts without writing")
	return c
}

// readRescopeItems reads "<memory-id> <scope>" lines, separated by tabs or
// spaces. Blank lines and lines starting with # are skipped.
func readRescopeItems(path string) ([]app.RescopeItem, error) {
	var raw []byte
	var err error
	if path == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, err
	}
	var items []app.RescopeItem
	for n, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Fields(line)
		if len(f) != 2 {
			return nil, fmt.Errorf("%s line %d: want '<memory-id> <scope>', got %q", path, n+1, line)
		}
		items = append(items, app.RescopeItem{Ref: f[0], Target: f[1]})
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s: no moves found", path)
	}
	return items, nil
}

// cliTrigger is the trigger stated with remember's flags, nil for none.
func cliTrigger(when, whenNot string, scopes, paths, tools []string) *trigger.Trigger {
	t := &trigger.Trigger{When: when, WhenNot: whenNot}
	if len(scopes)+len(paths)+len(tools) > 0 {
		t.Case = &trigger.Case{Scope: scopes, Paths: paths, Tools: tools}
	}
	if trigger.Empty(t) {
		return nil
	}
	return t
}

func showCmd() *cobra.Command {
	var view viewFlags
	c := &cobra.Command{Use: "show <memory-id>", Short: "Show memory text and its upstream mapping", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			units, err := view.binding(a).ResolveAll(a, args[0])
			if err != nil {
				return err
			}
			canonical := units[0]
			fmt.Printf("[%s] %s\ncategory: %s\n", canonical.SID, canonical.Body, canonical.Category)
			if t, err := trigger.Decode(canonical.Trigger); err == nil {
				for _, l := range trigger.Lines(t) {
					fmt.Println(l)
				}
			}
			if rs := a.Recalled(canonical.SID); rs.Count > 0 {
				fmt.Printf("recalled: %d times in %d sessions, last %s\n", rs.Count, rs.Sessions, rs.Last.Format("2006-01-02"))
			} else if since := a.RecallLogSince(); !since.IsZero() {
				fmt.Printf("recalled: never (logged since %s)\n", since.Format("2006-01-02"))
			}
			for _, u := range units {
				o, err := a.Origin(u)
				if err != nil {
					return err
				}
				fmt.Printf("source: %s\npath: %s\nkey: %s\norigin: %s\n", o.Source, o.Path, o.Key, o.Origin)
				if u.ID != canonical.ID && u.ContentHash != canonical.ContentHash {
					fmt.Printf("mirror text: %s\n", u.Body)
				}
			}
			return nil
		})
	}}
	view.flags(c)
	return c
}

func listCmd() *cobra.Command {
	var limit int
	var asJSON bool
	var view viewFlags
	c := &cobra.Command{Use: "list", Short: "List indexed memories", RunE: func(*cobra.Command, []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			b := view.binding(a)
			where, params := b.SQL("u.scope", "u.source_id")
			rows, err := a.DB.QueryContext(ctx, `SELECT sid, kind, `+b.ScopeExpr("u.scope", "u.source_id")+`, category, body, COALESCE(NULLIF(u.mtype,''), (SELECT t.mtype FROM inferred_types t WHERE t.content_hash=u.content_hash), ''), tags, updated_ns FROM units u WHERE (`+where+`) AND active=1 AND sid NOT IN (SELECT sid FROM suppressed)
    AND u.id=(SELECT c.id FROM units c JOIN sources cs ON cs.id=c.source_id WHERE c.sid=u.sid ORDER BY (cs.kind='native') DESC,c.updated_ns DESC,c.id LIMIT 1)
    ORDER BY updated_ns DESC LIMIT ?`, append(params, limit)...)
			if err != nil {
				return err
			}
			defer rows.Close()
			out := memoriesJSON{Memories: []memoryJSON{}}
			for rows.Next() {
				var u store.Unit
				var tags string
				var updated int64
				if err := rows.Scan(&u.SID, &u.Kind, &u.Scope, &u.Category, &u.Body, &u.Type, &tags, &updated); err != nil {
					return err
				}
				u.Tags, u.Updated = strings.Fields(tags), time.Unix(0, updated).UTC()
				if asJSON {
					out.Memories = append(out.Memories, unitJSON(u))
					continue
				}
				fmt.Printf("[%s] (%s, %s, %s) %s\n", u.SID, u.Kind, u.Scope, u.Category, recall.Truncate(strings.Join(strings.Fields(u.Body), " "), 200))
			}
			if err := rows.Err(); err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(out)
			}
			return nil
		})
	}}
	c.Flags().IntVarP(&limit, "limit", "n", 50, "max rows")
	c.Flags().BoolVar(&asJSON, "json", false, "JSON output: {\"memories\": [...]}, each with id, kind, scope, category, type, tags, text, updated")
	view.flags(c)
	return c
}

// memoriesJSON is the one shape of `hug mem list --json` and `search --json`.
type memoriesJSON struct {
	Memories []memoryJSON `json:"memories"`
}

type memoryJSON struct {
	ID       string           `json:"id"`
	Kind     string           `json:"kind"`
	Scope    string           `json:"scope"`
	Category string           `json:"category,omitempty"`
	Type     string           `json:"type,omitempty"`
	Trigger  *trigger.Trigger `json:"trigger,omitempty"`
	Tags     []string         `json:"tags,omitempty"`
	Text     string           `json:"text"`
	Updated  time.Time        `json:"updated"`
	Score    *float64         `json:"score,omitempty"`
}

func unitJSON(u store.Unit) memoryJSON {
	t, _ := trigger.Decode(u.Trigger)
	return memoryJSON{ID: u.SID, Kind: u.Kind, Scope: u.Scope, Category: u.Category, Type: u.Type, Trigger: t, Tags: u.Tags, Text: u.Body, Updated: u.Updated}
}

func relatedCmd() *cobra.Command {
	var view viewFlags
	c := &cobra.Command{Use: "related <unit-id>", Short: "Nearest neighbours of a memory", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				binding := view.binding(a)
				u, err := binding.Resolve(a, args[0])
				if err != nil {
					return err
				}
				res, err := a.Searcher.RelatedIn(ctx, u.ID, 10, binding.Scopes, binding.ScopeFor)
				if err != nil {
					return err
				}
				printResults(res, false)
				return nil
			})
		}}
	view.flags(c)
	return c
}

func statsCmd() *cobra.Command {
	var view viewFlags
	var fixes, asJSON bool
	var since string
	c := &cobra.Command{Use: "stats", Short: "Index statistics", RunE: func(*cobra.Command, []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			if fixes {
				// The fix trail (rjk44w): recall blocks, and the forgets and
				// corrections that followed, per UTC day.
				var from time.Time
				if since != "" {
					t, err := time.Parse("2006-01-02", since)
					if err != nil {
						return fmt.Errorf("--since %q is not a date\nfix: hug mem stats --fixes --since 2026-11-16", since)
					}
					from = t
				}
				s, err := app.SummarizeFixes(ctx, a, from)
				if err != nil {
					return err
				}
				if asJSON {
					return json.NewEncoder(os.Stdout).Encode(s)
				}
				fmt.Print(app.FixReport(s))
				return nil
			}
			if since != "" {
				return fmt.Errorf("--since applies to --fixes only\nfix: hug mem stats --fixes --since %s", since)
			}
			s, err := app.Stats(ctx, a, view.binding(a))
			if err != nil {
				return err
			}
			if l := harness.StaleList(staleMCPScan(ctx).Stale); l != "" {
				s.StaleMCP = l + " (older than v" + harness.ReloadSince + ", cannot reload; restart each in its app: jcode mcp action=reload server=hug, or quit and reopen the app)"
			}
			b, _ := json.MarshalIndent(s, "", "  ")
			fmt.Println(string(b))
			return nil
		})
	}}
	c.Flags().BoolVar(&fixes, "fixes", false, "the memory fix trail: recall blocks, forgets, reinforces, supersedes and rescopes per UTC day, whether recall showed the memory first, undone forgets, and the last 20 fixes")
	c.Flags().StringVar(&since, "since", "", "with --fixes: from this UTC day (YYYY-MM-DD)")
	c.Flags().BoolVar(&asJSON, "json", false, "with --fixes: JSON output")
	view.flags(c)
	return c
}

func gcCmd() *cobra.Command {
	return &cobra.Command{Use: "gc", Short: "Drop vectors from old pipelines or deleted units", RunE: func(*cobra.Command, []string) error {
		return withApp(func(ctx context.Context, a *app.App) error {
			if a.Embedder == nil {
				return fmt.Errorf("embedder unavailable; refusing to gc without knowing the current pipeline")
			}
			n, err := a.DB.GC(index.PipelineHash(a.Embedder.Model()))
			fmt.Println("removed", n, "vectors")
			return err
		})
	}}
}

// hookCmd is the harness entry point: `muninn hook claude|codex|jcode`.
func hookCmd() *cobra.Command {
	var budget time.Duration
	c := &cobra.Command{Use: "hook <claude|codex|jcode>", Short: "Run as a harness hook (reads the harness payload)", Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			// A hook must never break the harness: errors are logged, exit 0.
			// A jcode line names the session and event, so a timeout can be
			// tied to a turn (x8zzzn).
			if err := runHook(args[0], budget); err != nil {
				if sid := os.Getenv("JCODE_HOOK_SESSION_ID"); args[0] == "jcode" && sid != "" {
					logHook("jcode %s %s: %v", short(sid), os.Getenv("JCODE_HOOK_EVENT"), err)
				} else {
					logHook("%s: %v", args[0], err)
				}
			}
			return nil
		}}
	c.Flags().DurationVar(&budget, "budget", 8*time.Second, "time budget for recall")
	return c
}

func runHook(name string, budget time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	switch name {
	case "claude", "codex":
		in, err := harness.ReadPromptHook(os.Stdin)
		if err != nil {
			return err
		}
		resp, err := hookRecallBoard(ctx, name, in)
		if err != nil {
			return err
		}
		// A hug mcp server too old to reload itself (z77pem), once per
		// session, before what recall found: only the one serving this
		// session (the app this hook runs under).
		if in.SessionID != "" {
			resp.Block = withStaleMCP(staleMCPOnce(ctx, name+"-"+in.SessionID, ""), resp.Block)
		}
		os.Stdout.Write(harness.PromptHookOutput(resp.Block))
		return nil
	case "jcode":
		sid := os.Getenv("JCODE_HOOK_SESSION_ID")
		if sid == "" {
			return nil
		}
		// Each jcode instance runs with its own JCODE_HOME; the hook inherits
		// it. Use that instance's sessions, binary and memories only.
		jh := os.Getenv("JCODE_HOME")
		if jh == "" {
			home, _ := os.UserHomeDir()
			jh = filepath.Join(home, ".jcode")
		}
		sessions := filepath.Join(jh, "sessions")
		parks := harness.ParkDir(config.DataDir())
		// What hug delivered to this session (a person's request or answer,
		// gmzd87) is acknowledged once it is in the transcript: at the turn
		// it started, or at the first tool call after a soft interrupt.
		ackDelivered(ctx, jh, sid, os.Getenv("JCODE_HOOK_EVENT") == "post_tool_feedback")
		// What could not reach this session's name while it had no live
		// session (bbxdd7) is handed over at its turn's start, once; the
		// session acknowledges it at its next tool call, as above.
		if ev := os.Getenv("JCODE_HOOK_EVENT"); ev == "turn_start" || ev == "session_start" {
			deliverPendingHook(ctx, jh, sid)
		}
		switch os.Getenv("JCODE_HOOK_EVENT") {
		case "post_tool_feedback":
			if err := deliverParked(ctx, jh, parks, sessions, sid); err != nil {
				return err
			}
			return toolRecallHook(ctx, jh, sid)
		case "turn_end", "session_end":
			// What no tool call took is dropped: it was for this turn only.
			if harness.HasPark(parks, sid) {
				if p, ok, _ := harness.Claim(parks, sid); ok {
					if p.Block != "" {
						logHook("jcode %s: dropped=no-tool-call turn=%s", short(sid), p.Turn)
					}
					if p.Midturn {
						// Typed after the turn's last tool call: no tool
						// result carried its recall. Counted, never pushed.
						if tail, err := harness.ReadJcodeTail(sessions, sid, p.Offset); err == nil && tail.NewTurn(p.Turn) == "" {
							if n := len(tail.Pasted(p.Turn)); n > 0 {
								logHook("jcode %s: midturn turn=%s missed=%d", short(sid), p.Turn, n)
							}
						}
					}
				}
			}
			return nil
		}
		turn, err := harness.ReadJcodeTurn(sessions, sid)
		if err != nil {
			return err
		}
		msg := turn.Text
		a, err := app.Open(ctx)
		if err != nil {
			return err
		}
		defer a.Close()
		resp, inst, err := jcodeRecall(ctx, a, jh, sid, msg)
		// Mid-turn recall (x8zzzn): the park is the turn's anchor even with
		// no block, so the turn's tool calls can find what is typed into it.
		p := harness.Parked{Turn: turn.ID, Block: resp.Block}
		if err == nil && a.Cfg.JcodeMidturn() {
			if jt, end, ok := harness.JcodeTurnOf(sessions, sid, turn.ID); ok {
				p.Turn, p.Midturn, p.Offset = jt, true, end
			}
		}
		if err != nil || resp.Block == "" {
			logHook("jcode[%s] %s: msg=%q hits=0 skipped=%q elapsed=%s", inst.Name, short(sid), recall.Truncate(msg, 60), resp.Skipped, resp.Elapsed)
			if err != nil || !p.Midturn {
				return err
			}
			return parkLogged(parks, sid, p)
		}
		for _, h := range resp.Hits {
			p.Units = append(p.Units, h.UnitID)
		}
		// The block waits for this turn's first tool result (z9q42h): it
		// never goes into the person's input box, where jcode transcript
		// put it while he typed.
		err = parkLogged(parks, sid, p)
		logHook("jcode[%s] %s: msg=%q hits=%d elapsed=%s parked turn=%s err=%v", inst.Name, short(sid), recall.Truncate(msg, 60),
			len(resp.Hits), resp.Elapsed, turn.ID, err)
		return err
	default:
		return fmt.Errorf("unknown harness %q (claude, codex, jcode)", name)
	}
}

// jcodeRecall recalls for one message of jcode session sid: the turn's first
// message at turn_start, or one typed into the turn (mid-turn). Both take
// this one path, so what is recalled does not depend on when it arrives.
func jcodeRecall(ctx context.Context, a *app.App, jh, sid, msg string) (recall.Response, harness.JcodeInstance, error) {
	sessions := filepath.Join(jh, "sessions")
	inst := harness.InstanceForHome(a.JcodeInstances(), jh)
	binding := a.Bound(jh, os.Getenv("JCODE_HOOK_CWD"))
	if binding.Err != nil {
		return recall.Response{}, inst, binding.Err
	}
	cwd := os.Getenv("JCODE_HOOK_CWD")
	bs := boardSource(ctx, a.Cfg.BoardNudges(), cwd, sessionName(sid), jh, func(since time.Time) harness.Activity {
		act, _ := harness.JcodeActivity(sessions, sid, since)
		return act
	})
	resp, err := hookRecallWith(ctx, a, "jcode", sid, msg, binding, bs)
	if resp.BoardKind != "" {
		logHook("jcode[%s] %s: board=%s", inst.Name, short(sid), resp.BoardKind)
	}
	// A hug mcp server too old to reload itself (z77pem): checked once
	// per session, said with the session's first block at a tool result,
	// only for the server of this session's jcode home.
	if err == nil {
		resp.Block = withStaleMCP(staleMCPOnce(ctx, sid, jh), resp.Block)
	}
	return resp, inst, err
}

// parkLogged parks p, saying so in hook.log when it replaces a block no tool
// call or turn_end took (step 1 could not tell those from lost ones).
func parkLogged(parks, sid string, p harness.Parked) error {
	if old, ok := harness.Peek(parks, sid); ok && old.Block != "" {
		logHook("jcode %s: replaced parked=%s by=%s", short(sid), old.Turn, p.Turn)
	}
	return harness.Park(parks, sid, p)
}

// deliverPendingHook hands session sid what waited for its name (bbxdd7).
func deliverPendingHook(ctx context.Context, home, sid string) {
	got, err := huginn.DeliverPending(ctx, home, sid, os.Getenv)
	if err != nil {
		logHook("jcode %s: deliver pending: %v", short(sid), err)
	}
	for _, g := range got {
		logHook("jcode %s: delivered pending %s", short(sid), g)
	}
}

// ackDelivered acknowledges what hug delivered to session sid (gmzd87);
// in a tool's feedback it says so to the model, one line per item.
func ackDelivered(ctx context.Context, home, sid string, say bool) {
	acked, err := huginn.Acknowledge(ctx, home, sid)
	if err != nil {
		logHook("jcode %s: acknowledge: %v", short(sid), err)
	}
	for _, a := range acked {
		logHook("jcode %s: acknowledged %s", short(sid), a)
		if say {
			fmt.Fprintf(os.Stdout, "hug: you received %s; it now shows as acknowledged (hug task requests)\n", a)
		}
	}
}

// feedbackProbe, when set by a test, hears every load deliverParked does
// beyond its one stat.
var feedbackProbe func(what string)

// deliverParked is jcode's post_tool_feedback hook: it prints the session's
// parked recall block, which jcode appends to the tool result. With nothing
// parked, the cost is one stat. A block parked for another turn (the person
// has typed again, or the turn answered) is dropped, never shown.
//
// A mid-turn park (recall.jcode_midturn, x8zzzn) judges the turn by jcode's
// own turn id instead: a message typed into the running turn does not make
// the block stale. The tool result carries the parked block first, then the
// recall of each message typed into the turn since the last read; the park
// is put back, emptied, for the turn's later tool calls.
func deliverParked(ctx context.Context, jh, parks, sessions, sid string) error {
	if !harness.HasPark(parks, sid) {
		return nil
	}
	if feedbackProbe != nil {
		feedbackProbe("claim")
	}
	p, ok, err := harness.Claim(parks, sid)
	if err != nil || !ok {
		return err
	}
	if p.Midturn {
		return deliverMidturn(ctx, jh, parks, sessions, sid, p)
	}
	turn, err := harness.ReadJcodeTurn(sessions, sid)
	if err != nil || !turn.Current(p.Turn) {
		logHook("jcode %s: dropped=stale parked=%s current=%s ended=%v err=%v", short(sid), p.Turn, turn.ID, turn.Ended, err)
		return nil
	}
	printParked(sid, p)
	return nil
}

// printParked prints p's block into the tool result and marks its memories
// delivered.
func printParked(sid string, p harness.Parked) {
	fmt.Fprintln(os.Stdout, p.Block)
	logHook("jcode %s: delivered turn=%s tool=%s", short(sid), p.Turn, os.Getenv("JCODE_HOOK_TOOL_NAME"))
	if len(p.Units) == 0 {
		return
	}
	a, err := app.OpenAudit()
	if err != nil {
		logHook("jcode %s: delivered_ns: %v", short(sid), err)
		return
	}
	defer a.Close()
	if err := a.DB.MarkDelivered("jcode:"+sid, p.Units, time.Now()); err != nil {
		logHook("jcode %s: delivered_ns: %v", short(sid), err)
	}
}

// deliverMidturn is deliverParked for a mid-turn park.
func deliverMidturn(ctx context.Context, jh, parks, sessions, sid string, p harness.Parked) error {
	tail, err := harness.ReadJcodeTail(sessions, sid, p.Offset)
	if err != nil {
		if p.Block != "" {
			logHook("jcode %s: dropped=stale parked=%s current= ended=false err=%v", short(sid), p.Turn, err)
		}
		return nil
	}
	if nt := tail.NewTurn(p.Turn); nt != "" {
		// A newer turn began: what was parked, and what was typed into the
		// old turn, belongs to the past. The new turn parks its own.
		if p.Block != "" {
			logHook("jcode %s: dropped=stale parked=%s current=%s ended=true err=<nil>", short(sid), p.Turn, nt)
		}
		return nil
	}
	if p.Block != "" {
		printParked(sid, p)
	}
	if pasted := tail.Pasted(p.Turn); len(pasted) > 0 {
		if err := recallPasted(ctx, jh, sid, p.Turn, pasted); err != nil {
			logHook("jcode %s: midturn turn=%s err=%v", short(sid), p.Turn, err)
		}
	}
	// Put back, emptied, unless a newer turn_start parked meanwhile.
	_, err = harness.ParkIfAbsent(parks, sid, harness.Parked{Turn: p.Turn, Midturn: true, Offset: tail.End})
	return err
}

// recallPasted recalls each message typed into turn and prints its block
// into this tool result, as turn_start does for the turn's first message.
func recallPasted(ctx context.Context, jh, sid, turn string, pasted []harness.JcodeMsg) error {
	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()
	for _, m := range pasted {
		resp, inst, err := jcodeRecall(ctx, a, jh, sid, m.Text)
		if err != nil {
			return err
		}
		if resp.Block != "" {
			fmt.Fprintln(os.Stdout, resp.Block)
			var units []int64
			for _, h := range resp.Hits {
				units = append(units, h.UnitID)
			}
			if err := a.DB.MarkDelivered("jcode:"+sid, units, time.Now()); err != nil {
				logHook("jcode %s: delivered_ns: %v", short(sid), err)
			}
		}
		logHook("jcode[%s] %s: midturn turn=%s msg=%s hits=%d elapsed=%s skipped=%q tool=%s", inst.Name, short(sid), turn, m.ID,
			len(resp.Hits), resp.Elapsed, resp.Skipped, os.Getenv("JCODE_HOOK_TOOL_NAME"))
	}
	return nil
}

// toolRecallHook is the second half of jcode's post_tool_feedback hook
// (n7jbmx): with tool_recall on it prints the memories that name a path or
// command of the call just made. It opens the store without an embedder and
// never reindexes: it runs after every tool call.
func toolRecallHook(ctx context.Context, jh, sid string) error {
	cfg, err := config.Load()
	if err != nil || !cfg.ToolRecalls() {
		return nil
	}
	tool := os.Getenv("JCODE_HOOK_TOOL_NAME")
	var touched []string
	for _, p := range strings.Split(os.Getenv("JCODE_HOOK_TOUCHED_PATHS"), "\n") {
		if p = strings.TrimSpace(p); p != "" {
			touched = append(touched, p)
		}
	}
	var command string
	if tool == "bash" {
		var in struct {
			Command string `json:"command"`
		}
		json.Unmarshal([]byte(os.Getenv("JCODE_HOOK_TOOL_INPUT")), &in)
		command = in.Command
	}
	if command == "" && len(touched) == 0 {
		return nil
	}
	a, err := app.OpenAudit()
	if err != nil {
		return err
	}
	defer a.Close()
	cwd := os.Getenv("JCODE_HOOK_CWD")
	b := a.Bound(jh, cwd)
	if b.Err != nil {
		return b.Err
	}
	home, _ := os.UserHomeDir()
	e := &recall.Engine{DB: a.DB}
	resp, err := e.ToolRecall(ctx, recall.ToolRequest{Session: "jcode:" + sid, Harness: "jcode", Tool: tool, Command: command, Touched: touched,
		Root: cwd, Home: home, Scopes: b.Scopes, ScopeFor: b.ScopeFor, Similar: toolSimilar(a)}.Shipped())
	if err != nil {
		logHook("jcode %s: tool=%s err=%v", short(sid), tool, err)
		return err
	}
	if resp.Block != "" {
		fmt.Fprintln(os.Stdout, resp.Block)
	}
	logHook("jcode %s: tool=%s hits=%d elapsed=%s skipped=%q", short(sid), tool, len(resp.Hits), resp.Elapsed, resp.Skipped)
	return nil
}

// toolSimilar is the cosine gate of tool-call recall: it picks the embedder
// only when an anchor already matched, so most tool calls never touch it. A
// test replaces it.
var toolSimilar = func(a *app.App) func(context.Context, string, []int64) (map[int64]float64, error) {
	return func(ctx context.Context, text string, ids []int64) (map[int64]float64, error) {
		em, note := app.PickEmbedder(ctx, a.Cfg)
		if em == nil {
			return nil, errors.New(note)
		}
		return (&search.Searcher{DB: a.DB, Embedder: em}).Similarities(ctx, text, ids)
	}
}

func hookRecall(ctx context.Context, harnessName, session, msg, cwd string) (recall.Response, error) {
	a, err := app.Open(ctx)
	if err != nil {
		return recall.Response{}, err
	}
	defer a.Close()
	b := a.Bound("", cwd)
	if b.Err != nil {
		return recall.Response{}, b.Err
	}
	// Each prompt (and so each session start) brings in other clones'
	// project memories when the last pass is stale; never waits (0hkaa5).
	project.NudgeIfStale(cwd)
	return hookRecallWith(ctx, a, harnessName, session, msg, b, nil)
}

// hookRecallBoard is hookRecall for a Claude Code / Codex payload, with the
// board nudge reading tool calls from the payload's transcript (qawmnb).
func hookRecallBoard(ctx context.Context, harnessName string, in harness.PromptHookInput) (recall.Response, error) {
	a, err := app.Open(ctx)
	if err != nil {
		return recall.Response{}, err
	}
	defer a.Close()
	b := a.Bound("", in.Cwd)
	if b.Err != nil {
		return recall.Response{}, b.Err
	}
	var bs recall.BoardSource
	if in.TranscriptPath != "" {
		bs = boardSource(ctx, a.Cfg.BoardNudges(), in.Cwd, "", "", func(since time.Time) harness.Activity {
			act, _ := harness.ClaudeActivity(in.TranscriptPath, since)
			return act
		})
	}
	project.NudgeIfStale(in.Cwd) // as hookRecall (0hkaa5)
	resp, err := hookRecallWith(ctx, a, harnessName, in.SessionID, in.Prompt, b, bs)
	if resp.BoardKind != "" {
		logHook("%s %s: board=%s", harnessName, short(in.SessionID), resp.BoardKind)
	}
	return resp, err
}

func hookRecallWith(ctx context.Context, a *app.App, harnessName, session, msg string, b app.Binding, board recall.BoardSource) (recall.Response, error) {
	if _, err := ensureFresh(ctx, a); err != nil {
		logHook("reindex: %v", err)
	}
	b = b.Refresh(a)
	return a.Recall.Recall(ctx, recall.Request{Session: harnessName + ":" + session, Harness: harnessName, Message: msg, Scopes: b.Scopes, ScopeFor: b.ScopeFor, DreamOpen: func() int { return b.OpenDreamCount(a) }, Board: board})
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func short(s string) string {
	if len(s) > 24 {
		return s[:24]
	}
	return s
}

// LogHook appends one line to the memory store's hook.log: how a hook
// reports a problem without breaking the harness (cmd/hug's refusal of an
// unknown variable uses it, a0p5kt).
func LogHook(format string, args ...any) { logHook(format, args...) }

func init() {
	index.OnBackfill = func(units int, took time.Duration) {
		logHook("anchors: recorded for %d memories in %s (once, tool-call recall)", units, took.Round(time.Millisecond))
	}
}

func logHook(format string, args ...any) {
	p := filepath.Join(config.DataDir(), "hook.log")
	os.MkdirAll(filepath.Dir(p), 0o700)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s "+format+"\n", append([]any{time.Now().Format(time.RFC3339)}, args...)...)
}

func configCmd() *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Show paths and effective configuration", RunE: func(*cobra.Command, []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Println("config:", filepath.Join(config.ConfigDir(), "config.json"))
		fmt.Println("data:  ", config.DataDir())
		b, _ := json.MarshalIndent(cfg, "", "  ")
		fmt.Println(string(b))
		return nil
	}}
	c.AddCommand(&cobra.Command{Use: "init", Short: "Write a config.json with defaults", RunE: func(*cobra.Command, []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		return config.Save(cfg)
	}})
	return c
}

// jsonOutput are commands whose stdout is a JSON document.
var jsonOutput = map[string]bool{"stats": true, "config": true}

// nextHints names the usual next command after each top-level command.
var nextHints = map[string]string{
	"search":    "hug mem show ID (read one), hug mem forget ID (wrong or outdated)",
	"list":      "hug mem show ID, or hug mem search \"words\"",
	"show":      "hug mem related ID, or hug mem forget ID if it is wrong",
	"related":   "hug mem show ID",
	"remember":  "hug mem search \"words\" to check it is found",
	"forget":    "hug mem remember \"the corrected fact\" if it was wrong rather than obsolete",
	"rescope":   "hug mem list",
	"stats":     "hug mem tidy scan (placement and duplicates), or hug mem tui",
	"recall":    "hug mem show ID for any line above",
	"types":     "hug mem stats",
	"dream":     "hug mem tui",
	"index":     "hug mem stats",
	"gc":        "hug mem stats",
	"project":   "hug mem list",
	"config":    "hug mem stats",
	"sync":      "hug mem stats",
	"import":    "hug mem stats",
	"install":   "hug setup",
	"uninstall": "hug setup",
	"version":   "hug mem stats",
}

func nextHint(c *cobra.Command) string {
	top := topLevel(c)
	switch top {
	case "hook", "mcp", "serve", "tui", "capture", "jcode", "completion", "help":
		return ""
	}
	if f := c.Flags().Lookup("json"); f != nil && f.Value.String() == "true" {
		return ""
	}
	return nextHints[top]
}

// shared reports a scope whose memories sync with the project's other
// clones: a project's own, or a client project's.
func shared(scope string) bool {
	return strings.HasPrefix(scope, "project:") || strings.HasPrefix(scope, "client:")
}

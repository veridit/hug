// Package cli is the shared non-interactive command surface for CLI and MCP.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/veridit/hug/internal/env"
	"github.com/veridit/hug/internal/sharefile"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/editor"
	"github.com/veridit/hug/huginn/internal/harness"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/internal/cond"
)

const Help = `hug task: tasks on a Git branch, for humans and agents.
New here, or switching a repo from another tracker? Run: hug task setup
Use --json with any command. IDs, unique prefixes, slugs and aliases work.
  setup                     hug task setup
  init                      hug task init
  new                       hug task new "Fix retry" --type bug --goal "Retries work" --accept "Test passes"
  ls                        hug task ls --status todo --label api --mine
  show                      hug task show k7m2qx
  edit                      hug task edit k7m2qx
  move                      hug task move k7m2qx --before p3rt9a
  mv                        hug task mv k7m2qx auth
  take                      hug task take k7m2qx --model claude-opus-5-5
  done                      hug task done k7m2qx --reflect "the line 'X' in AGENTS.md made me Y"
  reflections               hug task reflections
  check                     hug task check k7m2qx 1
  strike                    hug task strike k7m2qx 2 --reason "replaced by the API test"
  note                      hug task note k7m2qx "Test evidence"
  dep                       hug task dep k7m2qx --add p3rt9a
  set                       hug task set k7m2qx --priority high --label api
  drop                      hug task drop k7m2qx
  next                      hug task next --type bug --priority high --group auth
  rebalance                 hug task rebalance
  types                     hug task types
  lint                      hug task lint
  sync                      hug task sync
  import                    hug task import backlog ./backlog
  tui                       hug task tui
  mcp                       hug task mcp
  install                   hug task install jcode --all-instances
  uninstall                 hug task uninstall jcode
  version                   hug task version
  help                      hug task help
New flags: --dep ID (repeatable), --group FOLDER, --label LABEL, --top, --after ID.
Details: hug task help COMMAND (for every flat flag and examples).
Who takes: --model MODEL (required; or HUG_MODEL; a person: --model human),
--as NAME (optional; or HUG_USER, or the agent session). Never git user.name.
Exit: 0 success, 1 error, 2 usage, 3 blocked, 4 no next task.
next: hug task next
`

var Commands = []string{"setup", "init", "new", "ls", "show", "edit", "move", "mv", "take", "done", "check", "strike", "promote", "note", "dep", "set", "drop", "wait", "resume", "design", "next", "rebalance", "types", "states", "lint", "sync", "import", "board", "tui", "mcp", "install", "uninstall", "upgrade", "attach", "detach", "reflections", "help"}

type UsageError struct{ Problem, Fix string }
type OperationError struct{ Problem, Fix string }

func (e *OperationError) Error() string { return e.Problem }

func (e *UsageError) Error() string   { return e.Problem }
func usage(problem, fix string) error { return &UsageError{problem, fix} }

type App struct {
	Cwd      string
	In       io.Reader
	Out, Err io.Writer
	// Version is what hug version prints; hug tui shows it in the key bar.
	Version string
	// Env overrides the process environment for this App's commands (who
	// takes, the model trailer, the editor, COLUMNS): a key present here wins,
	// "" included. Nil reads the process environment. Tests set it instead of
	// os.Setenv, so they can run in parallel (0sbvnq).
	Env map[string]string
}

// environ is the process environment with a.Env applied, for a child
// process that stands in for the user (the editor); nil when a.Env is
// empty, which exec reads as the process environment.
func (a *App) environ() []string {
	if len(a.Env) == 0 {
		return nil
	}
	out := make([]string, 0, len(os.Environ())+len(a.Env))
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); !hasKey(a.Env, k) {
			out = append(out, kv)
		}
	}
	for k, v := range a.Env {
		out = append(out, k+"="+v)
	}
	return out
}

// hasKey reports whether m holds k; on Windows variable names ignore case.
func hasKey(m map[string]string, k string) bool {
	if _, ok := m[k]; ok {
		return true
	}
	if runtime.GOOS == "windows" {
		for mk := range m {
			if strings.EqualFold(mk, k) {
				return true
			}
		}
	}
	return false
}

// getenv is name from a.Env, else from the process environment.
func (a *App) getenv(name string) string {
	if v, ok := a.Env[name]; ok {
		return v
	}
	return env.Get(name)
}

type Reply map[string]any

type gitCache struct{ r *gitbranch.Repository }

func (g gitCache) Head(context.Context) (string, error) { return g.r.Head() }
func (g gitCache) Diff(_ context.Context, a, b string) ([]cache.Change, error) {
	cs, e := g.r.DiffNameStatus(a, b)
	out := make([]cache.Change, 0, len(cs))
	for _, c := range cs {
		out = append(out, cache.Change{Status: c.Status, OldPath: c.OldPath, Path: c.Path})
	}
	return out, e
}

type session struct {
	r *gitbranch.Repository
	c *cache.Cache
	s *task.Store
	// nudged: this command started (or found) a background pass (0hkaa5).
	nudged bool
}

// Task branch layout (2026-11-15): hug keeps each kind of data on its own
// branch under .hug/. The old huginn layout is moved by "hug upgrade".
const (
	TaskBranch    = "hug/tasks"
	TaskDir       = ".hug/tasks"
	OldTaskBranch = "huginn"
	OldTaskDir    = ".huginn"
)

// oldLayout reports whether root still uses the pre-hug task layout.
func oldLayout(ctx context.Context, root string) bool {
	if _, e := os.Stat(filepath.Join(root, OldTaskDir, ".git")); e == nil {
		return true
	}
	// One git process for both refs (44w1ev). A for-each-ref pattern also
	// matches refs below it (refs/heads/huginn/x), so compare exactly.
	out, e := exec.CommandContext(ctx, "git", "-C", root, "for-each-ref", "--format=%(refname)",
		"refs/heads/"+OldTaskBranch, "refs/remotes/origin/"+OldTaskBranch).Output()
	if e != nil {
		return false
	}
	for _, ref := range strings.Split(string(out), "\n") {
		if ref == "refs/heads/"+OldTaskBranch || ref == "refs/remotes/origin/"+OldTaskBranch {
			return true
		}
	}
	return false
}

func (a *App) open(ctx context.Context) (*session, error) { return a.openBoard(ctx, false) }

// openBoard opens the board; lazy leaves the first read to the command
// (a write reads under the board turn, so a read before it is wasted: 44w1ev
// step 8, one walk of every task file).
func (a *App) openBoard(ctx context.Context, lazy bool) (*session, error) {
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	// One look at the repository serves the checks below and the board's
	// checkout (44w1ev: open spawned 13 git processes, ~15 ms each).
	loc, e := gitbranch.Locate(cwd)
	if e != nil {
		var ge *gitbranch.Error
		if errors.As(e, &ge) && strings.Contains(ge.Detail, "Git >= 2.42") {
			return nil, e
		}
		return nil, &OperationError{Problem: "hug task needs a Git checkout as its working directory", Fix: "cd /path/to/your/git/repo && hug task init"}
	}
	if oldLayout(ctx, loc.Main) {
		return nil, &OperationError{Problem: "tasks are still on the old layout (branch huginn at .huginn/); hug now keeps them on branch hug/tasks at .hug/tasks/",
			Fix: "hug upgrade"}
	}
	// One board per repository: a linked worktree (git worktree add) uses the
	// main worktree's hug/tasks checkout, never a second one (065wwn).
	r, e := loc.Open(TaskBranch, TaskDir)
	if e != nil {
		return nil, e
	}
	// Waits (16dxsr) are checked against this checkout: its branches and
	// tags, and this machine's load. Git is asked only when a wait asks.
	world := cond.Local(cwd, func() string { return gitbranch.DefaultBranch(cwd) })
	task.SetWorld(func() cond.State { return world })
	gd := r.Common
	c, e := cache.Open(r.Dir, filepath.Join(gd, "huginn", "cache.db"), gitCache{r})
	if e != nil {
		return nil, e
	}
	// One command reads the board once (44w1ev): List syncs, and the Store
	// keeps the result until it writes a task file itself.
	s := &task.Store{Root: r.Dir, Snapshot: true}
	s.Refresh = func() ([]task.Task, error) { return c.List(ctx) }
	// A write rereads only the files it wrote (44w1ev step 8).
	s.RefreshWritten = func(paths []string) ([]task.Task, error) { return c.ListAfterWrite(ctx, paths) }
	if !lazy {
		if _, e = s.Tasks(); e != nil {
			c.Close()
			return nil, e
		}
	}
	return &session{r: r, c: c, s: s}, nil
}
func (a *App) refreshExisting(ctx context.Context) error {
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	// The board lives in the main worktree, also seen from a linked one (065wwn).
	root, e := gitbranch.MainRoot(cwd)
	if e != nil {
		return nil
	}
	if _, e = os.Lstat(filepath.Join(root, TaskDir)); os.IsNotExist(e) {
		return nil
	} else if e != nil {
		return e
	}
	s, e := a.open(ctx)
	if e != nil {
		return e
	}
	return s.c.Close()
}

// Who takes a task (et85h1, decided by Jørgen 2026-11-16). The model is
// required: --model, otherwise HUG_MODEL; a person gives --model human. The
// name is optional: --as, otherwise HUG_USER, otherwise the harness session
// (jcode: the name in JCODE_SESSION_ID, session_<name>_...). Without a name
// the assignee is the model. git config user.name is never used: it names
// the person at the keyboard, not the agent doing the work.
type taker struct {
	Name, Model string
	// At is where the take happens (x1t1t1): detected from the working
	// directory, or --branch/--worktree.
	At task.Place
}

// harnessName is the session name the agent harness gives this process.
func harnessName(sessionID string) string {
	if id := strings.TrimSpace(sessionID); strings.HasPrefix(id, "session_") {
		if name, _, ok := strings.Cut(strings.TrimPrefix(id, "session_"), "_"); ok && name != "" {
			return name
		}
	}
	return ""
}

func (a *App) whoTakes(f flags) taker {
	t := taker{Name: strings.TrimSpace(f.get("as")), Model: strings.TrimSpace(f.get("model"))}
	t.At = placeOf(a.Cwd)
	if b := strings.TrimSpace(f.get("branch")); b != "" {
		t.At.Branch = b
	}
	if w := strings.TrimSpace(f.get("worktree")); w != "" {
		t.At.Worktree = w
	}
	if t.Model == "" {
		t.Model = a.takerModel(f)
	}
	if t.Name == "" {
		t.Name = strings.TrimSpace(a.getenv(env.User))
	}
	if t.Name == "" {
		t.Name = harnessName(a.getenv("JCODE_SESSION_ID"))
	}
	return t
}

// takerModel is the model a command runs as: --model, else HUG_MODEL.
func (a *App) takerModel(f flags) string {
	if m := strings.TrimSpace(f.get("model")); m != "" {
		return m
	}
	return strings.TrimSpace(a.getenv(env.Model))
}

// assignee is the name the board records for t: the name, else the model.
func (t taker) assignee() string {
	if t.Name != "" {
		return t.Name
	}
	return t.Model
}

// identity is who --mine means: the same name take records.
func (a *App) identity(f flags) (string, error) {
	if who := a.whoTakes(f).assignee(); who != "" {
		return who, nil
	}
	return "", usage("--mine needs to know who you are: no --as, HUG_USER, agent session or model given", "hug task next --mine --as NAME (or set HUG_USER or HUG_MODEL)")
}

type flags struct {
	values map[string][]string
	args   []string
}

func parse(args []string, allowed string, optionalValue ...string) (flags, error) {
	f := flags{values: map[string][]string{}}
	valid := strings.Fields(allowed + " json help")
	for i := 0; i < len(args); i++ {
		v := args[i]
		if v == "--" {
			f.args = append(f.args, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(v, "--") {
			f.args = append(f.args, v)
			continue
		}
		key, val, eq := strings.Cut(strings.TrimPrefix(v, "--"), "=")
		if !slices.Contains(valid, key) {
			e := task.Invalid("flag", key, valid)
			return f, usage("unknown flag --"+key+"; valid: --"+strings.Join(valid, ", --")+"; closest: --"+e.Closest, "hug task help")
		}
		if slices.Contains([]string{"json", "jsonl", "wide", "help", "mine", "top", "dry-run", "all-instances", "all", "replace", "summary", "undo", "clear", strings.TrimPrefix(SyncChildArg, "--")}, key) {
			if eq && val != "true" {
				return f, usage("--"+key+" is a boolean flag", "hug task help")
			}
			f.values[key] = []string{"true"}
			continue
		}
		if !eq {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				if slices.Contains(optionalValue, key) {
					// --code alone asks which codes there are (6neax1).
					i--
					f.values[key] = append(f.values[key], "")
					continue
				}
				return f, usage("missing value for --"+key, "hug task help")
			}
			val = args[i]
		}
		f.values[key] = append(f.values[key], val)
	}
	return f, nil
}
func (f flags) get(k string) string {
	v := f.values[k]
	if len(v) > 0 {
		return v[len(v)-1]
	}
	return ""
}
func (f flags) has(k string) bool { _, ok := f.values[k]; return ok }

// refCommands take a task ref as their first argument; refFlags take one
// as their value. Both accept task:ID, the form running text uses, and are
// turned into the bare id here so every reply and printed command shows
// the bare id and runs when pasted into a shell (xxvm6t).
var refCommands = []string{"show", "edit", "move", "mv", "take", "done", "drop", "wait", "resume", "design", "check", "strike", "promote", "note", "dep", "set", "attach", "detach"}
var refFlags = []string{"before", "after", "dep", "add", "remove", "parent", "epic"}

func bareRefs(cmd string, f *flags) {
	if cmd == "rebalance" {
		for i := range f.args {
			f.args[i] = task.BareRef(f.args[i])
		}
	} else if slices.Contains(refCommands, cmd) && len(f.args) > 0 {
		f.args[0] = task.BareRef(f.args[0])
	}
	for _, k := range refFlags {
		for i, v := range f.values[k] {
			f.values[k][i] = task.BareRef(v)
		}
	}
}
func (f flags) count(n int) error {
	if len(f.args) != n {
		return usage(fmt.Sprintf("expected %d positional arguments, got %d", n, len(f.args)), "hug task help")
	}
	return nil
}

func taskJSON(t task.Task, root string, ts []task.Task) Reply {
	b, _ := json.Marshal(t)
	out := Reply{}
	_ = json.Unmarshal(b, &out)
	out["model"] = t.Model
	out["goal"] = strings.TrimSpace(t.Goal)
	acceptance := []any{}
	for _, item := range t.Acceptance {
		// Three states (sfa8w4): open, ticked, struck (with the reason).
		a := map[string]any{"text": item.Text, "checked": item.Checked, "state": item.State()}
		if item.Struck {
			a["struck"], a["reason"] = true, item.Reason
			if item.StruckBy != "" {
				a["struck_by"] = item.StruckBy
			}
			if item.StruckOn != "" {
				a["struck_on"] = item.StruckOn
			}
		}
		acceptance = append(acceptance, a)
	}
	out["acceptance"] = acceptance
	deps := []string{}
	for _, d := range t.Dependencies {
		v, e := task.ResolveDependency(ts, d.Ref)
		if e == nil {
			deps = append(deps, v.ID)
		} else {
			deps = append(deps, d.Ref)
		}
	}
	out["dependencies"] = deps
	out["dependency_details"] = task.DependencyDetails(t, ts)
	// Conditions (16dxsr), checked now: met or not, and what hug saw.
	if ws := task.Waits(t, ts); len(ws) > 0 {
		out["waits"] = ws
	} else {
		delete(out, "waits")
	}
	children := []Reply{}
	for _, c := range childIndexFor(ts)[strings.ToLower(t.ID)] {
		children = append(children, Reply{"id": c.ID, "title": c.Title, "status": c.Status})
	}
	out["children"] = children
	out["notes"] = ""
	// Notes and Purpose, Principle and Relevant details (jnaspy) come from
	// the parse that made t: ls fills them from the cache in one query
	// (FillTexts), as ls at 10,000 tasks read and parsed every file again
	// for them (44w1ev). A Task without them (one command about one task,
	// or a file changed since the cache saw it) reads its file.
	texts := t.Texts
	if texts == nil || !texts.Bodies {
		if b, e := sharefile.ReadFile(filepath.Join(root, filepath.FromSlash(t.Path))); e == nil {
			texts = task.Parse(t.Path, b).Task.Texts
		}
	}
	if texts != nil {
		for _, sec := range task.DescriptionSections {
			out[descriptionKey(sec)] = texts.Description[sec]
		}
		out["notes"] = texts.Notes
	}
	for _, k := range []string{"assignee", "labels", "aliases", "acceptance", "dependency_details"} {
		if out[k] == nil {
			out[k] = []any{}
		}
	}
	return out
}
func resultJSON(r task.Result, s *session) (Reply, error) {
	if r.Task.ID == "" {
		return Reply{"changed": r.Changed, "next": r.Next, "renamed": r.Renamed}, nil
	}
	ts, e := s.s.Tasks()
	if e != nil {
		return nil, e
	}
	out := Reply{"task": taskJSON(r.Task, s.r.Dir, ts), "changed": r.Changed, "next": r.Next}
	if len(r.Warnings) > 0 {
		out["warnings"] = r.Warnings
	}
	if len(r.Renamed) > 0 {
		out["renamed"] = r.Renamed
		out["renames"] = r.Renames
	}
	return out, nil
}

// Execute performs exactly the same operation for shell and MCP callers.
func (a *App) Execute(ctx context.Context, args []string) (reply Reply, err error) {
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
		if cmd == "--help" || cmd == "--json" {
			args = append([]string{cmd}, args...)
			cmd = "help"
		}
	}
	if !slices.Contains(Commands, cmd) {
		e := task.Invalid("command", cmd, Commands)
		return nil, usage("unknown command "+cmd+"; valid: "+strings.Join(Commands, ", ")+"; closest: "+e.Closest, "hug task "+e.Closest+" --help")
	}
	allowed := map[string]string{"install": "all-instances jcode-home dry-run", "uninstall": "all-instances jcode-home dry-run", "new": "type priority goal purpose principle details accept dep group label top after", "ls": "status type priority label group mine jsonl epic as model", "board": "type priority label group mine all epic wide as model", "next": "type priority label group mine epic as model", "take": "as model branch worktree", "strike": "reason undo as model", "design": "clear as model for", "wait": "as model on", "done": "reflect as model", "reflections": "all", "tui": "as model", "move": "before after top", "dep": "add remove", "set": "priority type label status title assignee parent goal purpose principle details accept", "import": "dry-run cite", "show": "file", "lint": "code summary all", "sync": "background-pass", "attach": "name note replace safe-reason content"}
	var optional []string
	if cmd == "lint" {
		optional = []string{"code"}
	}
	f, e := parse(args, allowed[cmd], optional...)
	if e != nil {
		// A flag mistake is fixed by that command's help, not the index.
		var u *UsageError
		if errors.As(e, &u) && u.Fix == "hug task help" && cmd != "help" {
			u.Fix = "hug task " + cmd + " --help"
		}
		return nil, e
	}
	// "hug task lint help", before or after flags, asks for lint's help,
	// like --help (6neax1: it said "expected 0 positional arguments").
	// Only for commands without positional arguments: "new help" is a title.
	noArgs := []string{"lint", "ls", "next", "sync", "types", "states", "init", "board", "setup", "tui", "upgrade"}
	if slices.Contains(noArgs, cmd) && len(f.args) == 1 && f.args[0] == "help" {
		f.args = nil
		f.values["help"] = []string{"true"}
	}
	bareRefs(cmd, &f)
	if f.has("help") || cmd == "help" {
		if e = a.refreshExisting(ctx); e != nil {
			return nil, e
		}
		target := cmd
		if cmd == "help" {
			if len(f.args) > 1 {
				return nil, usage("help expects at most one command", "hug task help")
			}
			if len(f.args) == 0 {
				return Reply{"help": Help, "next": "hug task next"}, nil
			}
			target = f.args[0]
		}
		if !slices.Contains(Commands, target) {
			e := task.Invalid("command", target, Commands)
			return nil, usage("unknown command "+target+"; valid: "+strings.Join(Commands, ", ")+"; closest: "+e.Closest, "hug task help "+e.Closest)
		}
		return Reply{"help": commandHelp(target), "next": "hug task next"}, nil
	}
	if cmd == "states" {
		// The state and stage tree (6ephqk), from the one table in
		// task/states.go; reads no board.
		if e = f.count(0); e != nil {
			return nil, e
		}
		return Reply{"states": task.States, "transitions": task.Transitions, "states_text": task.StatesText(), "next": "hug task next"}, nil
	}
	if cmd == "types" {
		if e = a.refreshExisting(ctx); e != nil {
			return nil, e
		}
		if e = f.count(0); e != nil {
			return nil, e
		}
		return Reply{"types": task.TypeDescriptions, "next": "hug task next"}, nil
	}
	if cmd == "setup" {
		return a.runSetup(ctx, f)
	}
	if cmd == "upgrade" {
		return a.runUpgrade(ctx, f)
	}
	if cmd == "install" || cmd == "uninstall" {
		return a.runInstall(cmd, f)
	}
	if cmd == "sync" && f.has(strings.TrimPrefix(SyncChildArg, "--")) {
		return a.backgroundPass(ctx)
	}
	// Writes hold the board turn from reading to committing; take and sync
	// order their locks themselves (run, then turn), as a background pass does.
	writer := !readOnly[cmd] && cmd != "take" && cmd != "sync" && cmd != "tui" && cmd != "edit"
	s, e := a.openBoard(ctx, writer)
	if e != nil {
		return nil, e
	}
	defer s.c.Close()
	// Every commit hug makes on the board names the model that made it, the
	// same one take records (t709he).
	// Only the model: whoTakes also asks git where the take happens (three
	// git processes), which no other command needs (44w1ev budget, CI run
	// 44059170996: ls 4.84 s over 4.8 s under -race).
	if m := a.takerModel(f); m != "" {
		s.r.Trailers = []string{"Model: " + m}
	}
	if writer {
		defer s.turn()()
		// The first read happens under the turn: a background rebase
		// cannot move task files between this command reading and
		// committing them.
		s.s.Invalidate()
	}
	// Reads bring in other clones' changes: a background pass when the last
	// is stale (a lock try and a stat; it never waits on git or the network).
	defer s.nudgeIfStale()
	defer func() {
		if reply != nil && err == nil {
			s.syncStatus(reply)
		}
	}()
	empty := func() (Reply, error) {
		ts, e := s.s.Tasks()
		if e != nil {
			return nil, e
		}
		next := "hug task new \"title\" --goal \"outcome\" --accept \"observable check\""
		if r, e := task.Next(ts, task.Filter{}); e == nil {
			next = r.Next
		}
		return Reply{"next": next}, nil
	}
	filter := task.Filter{Type: f.get("type"), Priority: f.get("priority"), Label: f.get("label"), Group: f.get("group"), Status: f.get("status")}
	if ep := f.get("epic"); ep != "" {
		ts, e := s.s.Tasks()
		if e != nil {
			return nil, e
		}
		t, e := task.Resolve(ts, ep)
		if e != nil {
			return nil, e
		}
		filter.Epic = t.ID
	}
	if f.has("mine") {
		filter.Mine, e = a.identity(f)
		if e != nil {
			return nil, e
		}
	}
	if filter.Type != "" && !slices.Contains(task.Types, filter.Type) {
		e := task.Invalid("type", filter.Type, task.Types)
		e.Fix = "hug task " + cmd + " --type " + e.Closest
		if cmd == "set" && len(f.args) == 1 {
			if d, err := s.s.Get(f.args[0]); err == nil {
				e.ID = d.Task.ID
				e.Fix = "hug task set " + d.Task.ID + " --type " + e.Closest
			}
		}
		if cmd == "new" {
			e.Fix = `hug task new "title" --goal "outcome" --accept "observable check" --type ` + e.Closest
		}
		return nil, e
	}
	if filter.Status != "" && !slices.Contains(task.Statuses, filter.Status) {
		e := task.Invalid("status", filter.Status, task.Statuses)
		e.Fix = "hug task ls --status " + e.Closest
		if cmd == "set" && len(f.args) == 1 {
			if d, err := s.s.Get(f.args[0]); err == nil {
				e.ID = d.Task.ID
				e.Fix = "hug task set " + d.Task.ID + " --status " + e.Closest
			}
		}
		return nil, e
	}
	var r task.Result
	switch cmd {
	case "init":
		if e = f.count(0); e != nil {
			return nil, e
		}
		e = s.commit("hug task init")
		if e == nil {
			e = s.r.EnsureFirstCommit("hug task init")
		}
		if e != nil {
			return nil, e
		}
		// Say what exists now: branch, worktree and its first commit.
		reply, e := empty()
		head, _ := s.r.Head()
		remote := s.r.Remote
		if remote == "" {
			remote = "(none: tasks stay local until a remote exists)"
		}
		reply["message"] = fmt.Sprintf("tasks on branch %s, worktree %s, head %s, remote %s", s.r.Branch, s.r.Dir, shortHash(head), remote)
		reply["branch"], reply["worktree"], reply["head"] = s.r.Branch, s.r.Dir, head
		reply["next"] = "hug task new \"title\" --goal \"...\" --accept \"observable check\""
		return reply, e
	case "ls":
		if e = f.count(0); e != nil {
			return nil, e
		}
		ts, e := s.s.Tasks()
		if e != nil {
			return nil, e
		}
		if e = s.c.FillTexts(ctx, ts); e != nil {
			return nil, e
		}
		out := []Reply{}
		for _, t := range ts {
			if filter.Status != "" && t.Status != filter.Status || filter.Type != "" && t.Type != filter.Type || filter.Priority != "" && t.Priority != filter.Priority || filter.Group != "" && t.Group != filter.Group || filter.Label != "" && !slices.Contains(t.Labels, filter.Label) || filter.Mine != "" && !slices.Contains(t.Assignee, filter.Mine) || !task.InEpic(t, ts, filter.Epic) {
				continue
			}
			out = append(out, taskJSON(t, s.r.Dir, ts))
		}
		reply, e := empty()
		if len(out) > 0 {
			reply["next"] = "hug task show " + out[0]["id"].(string)
		}
		reply["tasks"] = out
		return reply, e
	case "show":
		if e = f.count(1); e != nil {
			return nil, e
		}
		if name := f.get("file"); name != "" {
			b, e := s.s.ReadFile(f.args[0], name)
			if e != nil {
				return nil, e
			}
			return Reply{"file": name, "content": string(b), "next": "hug task show " + f.args[0]}, nil
		}
		d, e := s.s.Get(f.args[0])
		if e != nil {
			return nil, e
		}
		ts, e := s.s.Tasks()
		if e != nil {
			return nil, e
		}
		_, files, e := s.s.Files(d.Task.ID)
		if e != nil {
			return nil, e
		}
		if files == nil {
			files = []task.File{}
		}
		return Reply{"task": taskJSON(d.Task, s.r.Dir, ts), "files": files, "next": task.NextHint(d.Task)}.withLint(s, d.Task), nil
	case "attach":
		if len(f.args) < 2 && !(len(f.args) == 1 && f.has("content")) {
			return nil, usage("attach needs a task and at least one file", `hug task attach ID FILE... [--name NAME] [--note "what this shows"] [--replace]`)
		}
		if f.has("name") && !f.has("content") && len(f.args) != 2 {
			return nil, usage("--name renames one file; attach one file at a time with --name", "hug task attach "+f.args[0]+" FILE --name NAME")
		}
		// Read and check every file first, so a refusal writes nothing.
		var ins []attachIn
		if f.has("content") {
			if f.get("name") == "" || len(f.args) != 1 {
				return nil, usage("--content needs --name and no file arguments", "hug task attach ID --content TEXT --name NAME")
			}
			b := []byte(f.get("content"))
			if e := task.CheckText(f.get("name"), b); e != nil {
				return nil, e
			}
			if hint := secretHint(b); hint != "" && f.get("safe-reason") == "" {
				return nil, &OperationError{Problem: f.get("name") + " looks like it may contain a secret (" + hint + ")", Fix: "remove it, or attach again with safe_reason"}
			}
			ins = append(ins, attachIn{f.get("name"), b})
		}
		for _, path := range f.args[1:] {
			ins2, e := readAttach(path, f.get("name"))
			if e != nil {
				return nil, e
			}
			for _, x := range ins2 {
				if e := task.CheckText(x.name, x.b); e != nil {
					return nil, e
				}
				if hint := secretHint(x.b); hint != "" && f.get("safe-reason") == "" {
					return nil, &OperationError{Problem: x.name + " looks like it may contain a secret (" + hint + ")",
						Fix: "remove it, or attach again with --safe-reason \"why this is safe\""}
				}
				ins = append(ins, x)
			}
		}
		var results []task.AttachResult
		for _, x := range ins {
			res, ar, e := s.s.Attach(f.args[0], x.name, x.b, f.get("note"), f.has("replace"))
			if e != nil {
				return nil, e
			}
			r = res
			results = append(results, ar)
		}
		if e = s.commit("hug task attach " + r.Task.ID); e != nil {
			return nil, e
		}
		reply, e := resultJSON(r, s)
		if e != nil {
			return nil, e
		}
		reply["attached"] = results
		reply["next"] = "hug task show " + r.Task.ID
		return reply, nil
	case "detach":
		if e = f.count(2); e != nil {
			return nil, e
		}
		r, e = s.s.Detach(f.args[0], f.args[1])
	case "next":
		if e = f.count(0); e != nil {
			return nil, e
		}
		r, e = s.s.Next(filter)
	case "lint":
		if len(f.args) > 1 {
			return nil, usage("lint takes at most one task", "hug task lint ID")
		}
		mode, code, e := lintFlags(f)
		if e != nil {
			return nil, e
		}
		fs, e := s.boardFindings(ctx, a.Cwd, a.getenv("JCODE_HOME"))
		if e != nil {
			return nil, e
		}
		// hug task lint ID: every finding of that task in detail, with the
		// one combined fix (fkpyg3), closed or not.
		if len(f.args) == 1 {
			d, e := s.s.Get(f.args[0])
			if e != nil {
				return nil, e
			}
			mine := forTask(fs, d.Task)
			if code != "" {
				mine = withCode(mine, code)
			}
			reply, e := empty()
			if e != nil {
				return nil, e
			}
			reply = lintReply(reply, mine, "detail", code)
			reply["next"] = task.NextHint(d.Task)
			if len(mine) > 0 {
				reply["next"] = task.CombinedFix(d.Task.ID, mine)
			}
			return reply, nil
		}
		// Closed tasks' findings are hidden unless --all (6neax1).
		var hidden []task.Finding
		if !f.has("all") {
			ts, e := s.s.Tasks()
			if e != nil {
				return nil, e
			}
			fs, hidden = task.OpenFindings(fs, ts)
		}
		if code != "" {
			fs, hidden = withCode(fs, code), withCode(hidden, code)
		}
		reply, e := empty()
		if e != nil {
			return nil, e
		}
		reply = lintReply(reply, fs, mode, code)
		setHidden(reply, hidden, code)
		return reply, nil
	case "sync":
		if e = f.count(0); e != nil {
			return nil, e
		}
		if e = s.syncNow(); e != nil {
			return nil, e
		}
		s.s.Invalidate() // the pull may have changed task files
		return empty()
	case "new":
		if e = f.count(1); e != nil {
			return nil, e
		}
		r, e = s.s.New(task.NewOptions{Title: f.args[0], Goal: f.get("goal"), Purpose: f.get("purpose"), Principle: f.get("principle"), Details: f.get("details"), Type: f.get("type"), Priority: f.get("priority"), Group: f.get("group"), Acceptance: f.values["accept"], Deps: f.values["dep"], Labels: f.values["label"], Top: f.has("top"), After: f.get("after"), Pattern: true})
	case "take":
		if e = f.count(1); e != nil {
			return nil, e
		}
		r, e = a.take(ctx, s, f.args[0], a.whoTakes(f))
	case "dep":
		if e = f.count(1); e != nil {
			return nil, e
		}
		if !f.has("add") && !f.has("remove") {
			return nil, usage("dep needs --add REF or --remove REF", "hug task dep "+f.args[0]+" --help")
		}
		r, e = changeDependencies(s.s, f.args[0], f.values["add"], f.values["remove"])
	case "done", "drop":
		if cmd == "drop" {
			// A reason is required (6ephqk): it becomes dropped_reason.
			if len(f.args) == 1 {
				return nil, usage("drop needs a reason: why this task will never be done", "hug task drop "+f.args[0]+` "duplicate of OTHER"`)
			}
			if e = f.count(2); e != nil {
				return nil, e
			}
		} else if e = f.count(1); e != nil {
			return nil, e
		}
		if cmd == "done" {
			r, e = s.s.Done(f.args[0])
			// The reflection is stored only once done succeeded (tb0dra):
			// a blocked done keeps the answer for the next try.
			if e == nil && strings.TrimSpace(f.get("reflect")) != "" && !task.NothingToReflect(f.get("reflect")) {
				who := a.whoTakes(f).assignee()
				if who == "" {
					who = strings.Join(r.Task.Assignee, ", ")
				}
				r, e = s.s.Note(r.Task.ID, task.ReflectionNote(f.get("reflect"), who, time.Now()))
			}
		} else {
			r, e = s.s.Drop(f.args[0], f.args[1])
		}
	case "wait":
		if len(f.args) == 1 {
			return nil, usage("wait needs what it waits on", "hug task wait "+f.args[0]+` "CI run 123"`)
		}
		if e = f.count(2); e != nil {
			return nil, e
		}
		on := f.args[1]
		if who := strings.TrimSpace(f.get("on")); who != "" {
			// Waiting on a person: "NAME: what", so the TUI shows it to
			// them as waiting: you (crab 12:38).
			on = who + ": " + on
		}
		r, e = s.s.Wait(f.args[0], on, a.whoTakes(f).assignee())
	case "resume":
		if e = f.count(1); e != nil {
			return nil, e
		}
		r, e = s.s.Resume(f.args[0])
	case "design":
		if f.has("clear") {
			if e = f.count(1); e != nil {
				return nil, e
			}
			r, e = s.s.Design(f.args[0], "", "", "")
		} else {
			if len(f.args) == 1 {
				return nil, usage("design needs the question, or --clear when it is answered", "hug task design "+f.args[0]+` "the question" --for NAME`)
			}
			if e = f.count(2); e != nil {
				return nil, e
			}
			r, e = s.s.Design(f.args[0], f.args[1], a.whoTakes(f).assignee(), f.get("for"))
		}
	case "reflections":
		return a.runReflections(s, f)
	case "check":
		if e = f.count(2); e != nil {
			return nil, e
		}
		n, err := strconv.Atoi(f.args[1])
		if err != nil || n < 1 {
			return nil, usage("acceptance index must be a positive integer", "hug task show "+f.args[0])
		}
		r, e = s.s.Check(f.args[0], n)
	case "strike":
		if e = f.count(2); e != nil {
			return nil, e
		}
		n, err := strconv.Atoi(f.args[1])
		if err != nil || n < 1 {
			return nil, usage("acceptance index must be a positive integer", "hug task show "+f.args[0])
		}
		if !f.has("undo") && strings.TrimSpace(f.get("reason")) == "" {
			return nil, usage("striking a check out needs a reason: why it no longer applies", "hug task strike "+f.args[0]+" "+f.args[1]+" --reason \"why it no longer applies\"")
		}
		r, e = s.s.Strike(f.args[0], n, f.get("reason"), a.whoTakes(f).assignee(), f.has("undo"))
	case "promote":
		if e = f.count(2); e != nil {
			return nil, e
		}
		n, err := strconv.Atoi(f.args[1])
		if err != nil || n < 1 {
			return nil, usage("acceptance index must be a positive integer", "hug task show "+f.args[0])
		}
		r, e = s.s.Promote(f.args[0], n)
		if e == nil {
			if e = s.commit("hug task promote " + f.args[0] + " " + f.args[1]); e != nil {
				return nil, e
			}
			reply, e := resultJSON(r, s)
			if e == nil {
				withGuide(reply, "promote", r.Task, s)
			}
			return reply, e
		}
	case "note":
		if e = f.count(2); e != nil {
			return nil, e
		}
		r, e = s.s.Note(f.args[0], f.args[1])
	case "set":
		if e = f.count(1); e != nil {
			return nil, e
		}
		fields := []string{"status", "type", "priority", "label", "title", "assignee", "parent"}
		count := 0
		for _, field := range append(append([]string{}, fields...), "goal", "purpose", "principle", "details", "accept") {
			if f.has(field) {
				count++
			}
		}
		if count == 0 {
			return nil, usage("set needs a field: --priority, --type, --label, --status, --title, --assignee, --parent, --goal, --purpose, --principle, --details, --accept", "hug task show "+f.args[0])
		}
		if f.has("title") && strings.TrimSpace(f.get("title")) == "" {
			return nil, usage("title must not be empty", "hug task show "+f.args[0])
		}
		if v := f.get("status"); v != "" && !slices.Contains(task.Statuses, v) {
			return nil, task.Invalid("status", v, task.Statuses)
		}
		for _, field := range fields {
			if f.has(field) {
				r, e = s.s.Set(f.args[0], field, f.get(field))
				if e != nil {
					break
				}
			}
		}
		// Description sections and acceptance in one write (fkpyg3): every
		// combined fix: line runs without an editor.
		if e == nil {
			sections := map[string]string{}
			for flag, name := range task.SectionFlags {
				if f.has(flag) {
					sections[name] = f.get(flag)
				}
			}
			if len(sections) > 0 || len(f.values["accept"]) > 0 {
				r, e = s.s.SetText(f.args[0], sections, f.values["accept"])
			}
		}
	case "move":
		if e = f.count(1); e != nil {
			return nil, e
		}
		if !f.has("before") && !f.has("after") && !f.has("top") {
			return nil, usage("move needs --before ID, --after ID or --top; use mv for a group change", "hug task move "+f.args[0]+" --help")
		}
		r, e = s.s.Move(f.args[0], task.MoveOptions{Before: f.get("before"), After: f.get("after"), Top: f.has("top")})
	case "mv":
		if e = f.count(2); e != nil {
			return nil, e
		}
		r, e = s.s.MoveGroup(f.args[0], f.args[1])
	case "rebalance":
		r, e = s.s.Rebalance(f.args)
	case "edit":
		if e = f.count(1); e != nil {
			return nil, e
		}
		d, err := s.s.Get(f.args[0])
		if err != nil {
			return nil, err
		}
		ed := editor.CommandFrom(a.getenv)
		c := exec.CommandContext(ctx, ed[0], append(ed[1:], filepath.Join(s.r.Dir, d.Task.Path))...)
		c.Env = a.environ()
		c.Stdin = a.In
		c.Stdout = a.Out
		if f.has("json") {
			c.Stdout = a.Err
		}
		c.Stderr = a.Err
		if e = c.Run(); e != nil {
			return nil, e
		}
		s.s.Invalidate() // the editor changed the file
		// The turn covers the commit, not the editor: a background rebase
		// meanwhile refuses to start over the edited file (uncommitted).
		release := s.turn()
		e = s.commit("hug task edit " + d.Task.ID)
		release()
		if e != nil {
			return nil, e
		}
		fs, err := s.c.Lint(ctx)
		if err != nil {
			return nil, err
		}
		reply, err := empty()
		if err != nil {
			return nil, err
		}
		reply = lintReply(reply, fs, "", "")
		reply["next"] = "hug task show " + d.Task.ID
		// The edited task's own first finding comes first: that is what the
		// edit broke (rx74bp).
		for _, x := range fs {
			if x.Path == d.Task.Path {
				reply["next"] = x.FixLine()
				break
			}
		}
		return reply, err
	case "import":
		return a.runImport(ctx, s, f)
	case "board":
		return a.runBoard(ctx, s, f, filter)
	case "tui":
		return a.runTUI(ctx, s, f)
	default:
		return nil, usage("command requires its dedicated transport", "hug task help")
	}
	if e != nil {
		var detail *task.Error
		if errors.As(e, &detail) && strings.Contains(detail.Fix, "<id>") {
			if cmd == "new" {
				detail.Fix = "hug task help new"
			} else if len(f.args) > 0 {
				if d, err := s.s.Get(f.args[0]); err == nil {
					detail.ID = d.Task.ID
					detail.Fix = strings.ReplaceAll(detail.Fix, "<id>", d.Task.ID)
				}
			}
		}
		return nil, e
	}
	if cmd == "rebalance" {
		// Rebalance touches filenames only: say so in one line instead of
		// printing the first task and every renamed id (2026-11-15: 2,172 ids
		// on seid-lang looked like damage). JSON keeps the full lists.
		msg := fmt.Sprintf("rebalanced: %d filenames rekeyed, task order and contents unchanged", len(r.Renamed))
		if len(r.Renamed) == 0 {
			msg = "rebalance: nothing to rekey"
		} else if e = s.commit(fmt.Sprintf("hug task rebalance (%d files)", len(r.Renamed))); e != nil {
			return nil, e
		}
		return Reply{"changed": r.Changed, "next": r.Next, "renamed": r.Renamed, "renames": r.Renames, "message": msg}, nil
	}
	if cmd != "next" && cmd != "take" {
		if e = s.commit("hug task " + cmd + " " + r.Task.ID); e != nil {
			return nil, e
		}
	}
	reply, e = resultJSON(r, s)
	if e == nil && r.Changed {
		withGuide(reply, cmd, r.Task, s)
	}
	// The task's own findings, every one, with one fix (fkpyg3): what was
	// just written or taken is checked by the same list lint uses.
	if e == nil && r.Task.ID != "" && (cmd == "new" || cmd == "set" || cmd == "take") {
		reply.withLint(s, r.Task)
	}
	// One question at done, never a form (tb0dra): asked when no answer
	// came with it, also on a repeated done.
	if e == nil && cmd == "done" && strings.TrimSpace(f.get("reflect")) == "" {
		g, _ := reply["guide"].([]string)
		reply["guide"] = append(g, "reflect: "+task.ReflectQuestion+". Answer: hug task done "+r.Task.ID+" --reflect \"...\"")
	}
	return reply, e
}

// withGuide adds what to do now for this state change (task.Guide).
func withGuide(reply Reply, cmd string, t task.Task, s *session) {
	action := cmd
	if cmd == "set" {
		switch t.Status {
		case "done":
			action = "done"
		case "dropped":
			action = "drop"
		case "doing":
			action = "take"
		}
	}
	ts, _ := s.s.Tasks()
	if g := task.Guide(action, t, ts); len(g) > 0 {
		reply["guide"] = g
	}
}

// ErrorFixes are the alternative fix: lines of err beyond ErrorDetails'
// one (n2hq8p: a refused take offers done, wait or back to todo); nil when
// there is only one way out.
func ErrorFixes(err error) []string {
	var t *task.Error
	if errors.As(err, &t) && len(t.Fixes) > 1 {
		return t.Fixes
	}
	return nil
}

func ErrorDetails(err error) (int, string, string) {
	code, problem, fix := 1, strings.ReplaceAll(err.Error(), "\n", "; "), "hug task help"
	var op *OperationError
	if errors.As(err, &op) {
		return 1, op.Problem, op.Fix
	}
	var u *UsageError
	if errors.As(err, &u) {
		return 2, u.Problem, u.Fix
	}
	var t *task.Error
	if errors.As(err, &t) {
		code = t.ExitCode()
		if t.Kind == "taken" {
			return 1, "taken by " + t.Value + " (" + t.ID + ")", "hug task next"
		}
		if t.Kind == "busy" {
			return code, "not taking " + t.ID + ": " + t.Value, t.Fix
		}
		fix = t.Fix
		problem = t.Kind
		if t.ID != "" {
			problem += " " + t.ID
		}
		if t.Field != "" {
			problem += " " + t.Field
		}
		if t.Value != "" {
			problem += ": " + t.Value
		}
		if len(t.Valid) > 0 {
			problem += "; valid: " + strings.Join(t.Valid, ", ") + "; closest: " + t.Closest
		}
		for _, m := range t.Matches {
			problem += "; " + m.Slug + " " + m.ID
		}
		fixes := []string{}
		for _, d := range t.Dependencies {
			problem += "; " + d.ID + " " + d.Title
			fixes = append(fixes, "hug task show "+d.ID)
		}
		for _, v := range t.Unchecked {
			problem += fmt.Sprintf("; %d. %s", v.Index, v.Text)
			fixes = append(fixes, v.Fix)
		}
		if len(fixes) > 0 && len(t.Cycle) == 0 {
			fix = strings.Join(fixes, " && ")
		}
	}
	var g *gitbranch.Error
	if errors.As(err, &g) {
		fix = g.Fix
	}
	var c *gitbranch.ConflictError
	if errors.As(err, &c) {
		fix = c.Fix
	}
	if fix == "" {
		fix = "hug task help"
	}
	problem = strings.ReplaceAll(strings.SplitN(problem, "; fix:", 2)[0], "\r", " ")
	return code, strings.ReplaceAll(problem, "\n", "; "), fix
}

// Text replies print as follows; JSON replies are encoded unchanged.
// takenBy is "poodle · claude-opus-5-5 · feat/x" for a task taken with --as
// and --model on branch feat/x, the model alone when the name is the model,
// the assignee alone for tasks taken before models were recorded, and empty
// when untaken. The branch is added when one was recorded (x1t1t1), except
// when it is hide (ls rows hide the main branch; show always names it).
func takenBy(t Reply, hide string) string {
	names := []string{}
	if as, ok := t["assignee"].([]any); ok {
		for _, v := range as {
			names = append(names, fmt.Sprint(v))
		}
	}
	model, _ := t["model"].(string)
	branch, _ := t["branch"].(string)
	who := strings.Join(names, ", ")
	switch {
	case who == "":
		return ""
	case model != "" && who != model:
		who += " · " + model
	}
	if branch != "" && branch != hide {
		who += " · " + branch
	}
	return who
}

func (a *App) Run(ctx context.Context, args []string) int {
	reply, e := a.Execute(ctx, args)
	jsonMode := isJSON(args)
	if e != nil {
		code, p, fix := ErrorDetails(e)
		p = strings.NewReplacer("\r", " ", "\n", "; ").Replace(p)
		fixes := ErrorFixes(e)
		if jsonMode {
			r := Reply{"error": p, "fix": fix, "next": fix, "exit_code": code}
			if fixes != nil {
				r["fixes"] = fixes
			}
			_ = json.NewEncoder(a.Out).Encode(r)
		} else {
			fmt.Fprintln(a.Err, p)
			if fixes == nil {
				fixes = []string{fix}
			}
			for _, f := range fixes {
				fmt.Fprintln(a.Err, "fix: "+f)
			}
		}
		return code
	}
	if jsonMode {
		_ = json.NewEncoder(a.Out).Encode(reply)
		return 0
	}
	if isJSONL(args) {
		// One task per line, for grep, head and jq -c pipelines.
		if ts, ok := reply["tasks"].([]Reply); ok {
			enc := json.NewEncoder(a.Out)
			for _, t := range ts {
				_ = enc.Encode(t)
			}
		}
		return 0
	}
	if rs, ok := reply["installations"].([]harness.InstallResult); ok {
		for _, r := range rs {
			fmt.Fprintf(a.Out, "%s changed=%t backup=%s dry-run=%v\n", r.File, r.Changed, r.Backup, reply["dry_run"])
		}
	}
	if c, ok := reply["content"].(string); ok {
		// show --file: the file itself, then the next hint (2026-11-16:
		// text mode printed only the hint, so agents could not read files).
		fmt.Fprint(a.Out, c)
		if c != "" && !strings.HasSuffix(c, "\n") {
			fmt.Fprintln(a.Out)
		}
	}
	if h, ok := reply["help"].(string); ok {
		fmt.Fprint(a.Out, h)
		return 0
	}
	if steps, ok := reply["setup"].([]SetupStep); ok {
		fmt.Fprint(a.Out, reply["intro"])
		fmt.Fprintf(a.Out, "Checklist for this repository (%d open):\n", reply["open"])
		for _, s := range steps {
			mark := "[x]"
			if !s.Done {
				mark = "[ ]"
			}
			fmt.Fprintf(a.Out, "%s %s: %s\n", mark, s.Step, s.Detail)
			if s.Do != "" {
				fmt.Fprintf(a.Out, "    do: %s\n", strings.ReplaceAll(s.Do, "\n", "\n        "))
			}
		}
	}
	if cols, ok := reply["board"].([]BoardColumn); ok {
		if slices.Contains(args, "--wide") {
			writeBoardWide(a.Out, cols)
		} else {
			writeBoard(a.Out, cols, boardWidth(a.getenv("COLUMNS")))
		}
	}
	if ts, ok := reply["tasks"].([]Reply); ok {
		// Fixed-width columns first, the free-text title last, so the
		// id is always field 1 (2026-11-15, user: "id type slug desc").
		w := 0
		for _, t := range ts {
			w = min(40, max(w, len(fmt.Sprint(t["slug"]))))
		}
		main := gitbranch.DefaultBranch(a.Cwd)
		for _, t := range ts {
			// Who is working: only doing rows name the taker, so done history
			// does not crowd the list (show always names it), and the branch
			// only when it is not the main one.
			by := ""
			if who := takenBy(t, main); who != "" && t["status"] == "doing" {
				by = "  [" + who + "]"
			}
			// Acceptance progress, ticked/total (d8yx7y); "-" without checks.
			// After type, so the id stays field 1.
			checks := "-"
			if items, ok := t["acceptance"].([]any); ok && len(items) > 0 {
				done := 0
				for _, v := range items {
					if m, ok := v.(map[string]any); ok && m["checked"] == true {
						done++
					}
				}
				checks = fmt.Sprintf("%d/%d", done, len(items))
			}
			fmt.Fprintf(a.Out, "%-6s  %-7s  %-7s  %-5s  %-*s  %s%s\n", t["id"], t["status"], t["type"], checks, w, t["slug"], t["title"], by)
		}
	}
	if t, ok := reply["task"].(Reply); ok {
		fmt.Fprintf(a.Out, "%s  %s  %s  %s\n", t["slug"], t["title"], t["id"], t["status"])
		if who := takenBy(t, ""); who != "" {
			fmt.Fprintf(a.Out, "taken by: %s\n", who)
		}
		fmt.Fprintf(a.Out, "goal: %s\n", t["goal"])
		for _, sec := range task.DescriptionSections {
			if v, _ := t[descriptionKey(sec)].(string); v != "" {
				fmt.Fprintf(a.Out, "%s: %s\n", strings.ToLower(sec), v)
			}
		}
		for i, v := range t["acceptance"].([]any) {
			m := v.(map[string]any)
			mark, text := " ", fmt.Sprint(m["text"])
			if m["checked"] == true {
				mark = "x"
			}
			if m["struck"] == true {
				// Plain text has no strikethrough: [-] and the decision.
				mark = "-"
				text = "~~" + text + "~~ (not relevant: " + fmt.Sprint(m["reason"])
				for _, k := range []string{"struck_by", "struck_on"} {
					if v, ok := m[k].(string); ok && v != "" {
						text += ", " + v
					}
				}
				text += ")"
			}
			fmt.Fprintf(a.Out, "%d. [%s] %s\n", i+1, mark, text)
		}
		if t["notes"] != "" {
			fmt.Fprintf(a.Out, "notes: %s\n", t["notes"])
		}
		if ds, ok := t["dependency_details"].([]task.DependencyDetail); ok {
			for _, d := range ds {
				status := d.Status
				if d.OpenChildren > 0 {
					status += fmt.Sprintf(", epic: %d open", d.OpenChildren)
				}
				fmt.Fprintf(a.Out, "dependency: %s  %s  [%s]  %s\n", d.ID, d.Title, status, d.Path)
			}
		}
		if ws, ok := t["waits"].([]task.WaitDetail); ok {
			for _, w := range ws {
				state := "waiting"
				if w.Met {
					state = "met"
				}
				line := fmt.Sprintf("waits: when %s  [%s: %s]", w.Text, state, w.Seen)
				if w.Wakes != "" {
					line += "  wakes " + w.Wakes
				}
				fmt.Fprintln(a.Out, line)
			}
		}
		if p, ok := t["parent"].(string); ok && p != "" {
			fmt.Fprintf(a.Out, "epic: %s\n", p)
		}
		if cs, ok := t["children"].([]Reply); ok {
			for _, c := range cs {
				fmt.Fprintf(a.Out, "child: %s  %s  [%s]\n", c["id"], c["title"], c["status"])
			}
		}
	}
	if rs, ok := reply["reflections"].([]task.Reflection); ok {
		if len(rs) == 0 {
			fmt.Fprintln(a.Out, "no open reflections")
		}
		for _, r := range rs {
			who := ""
			if r.Who != "" {
				who = " " + r.Who
			}
			fmt.Fprintf(a.Out, "%s %d  %s  %s%s  %s\n", r.ID, r.N, r.State, r.Date, who, clip(r.Title, 60))
			if r.Instruction != "" {
				fmt.Fprintf(a.Out, "  instruction: %q\n", r.Instruction)
			}
			fmt.Fprintf(a.Out, "  %s\n", r.Text)
		}
	}
	if fs, ok := reply["findings"].([]task.Finding); ok {
		sum, _ := reply["summary"].(LintSummary)
		writeFindings(a.Out, fs, sum)
	}
	if text, ok := reply["states_text"].(string); ok {
		fmt.Fprint(a.Out, text)
	}
	if l, ok := reply["lint"].(*TaskLint); ok {
		writeTaskLint(a.Out, l)
	}
	if ts, ok := reply["types"].([]task.TypeInfo); ok {
		for _, v := range ts {
			fmt.Fprintf(a.Out, "%s  %s\n", v.Value, v.Description)
		}
	}
	if gs, ok := reply["guide"].([]string); ok {
		for _, g := range gs {
			fmt.Fprintln(a.Out, "now: "+g)
		}
	}
	if us, ok := reply["upgrade"].([]string); ok {
		for _, u := range us {
			fmt.Fprintln(a.Out, u)
		}
		if d, ok := reply["diff"].(string); ok {
			fmt.Fprint(a.Out, d)
		}
	}
	if ws, ok := reply["warnings"].([]string); ok {
		for _, v := range ws {
			fmt.Fprintln(a.Out, "warning: "+v)
		}
	}
	// The last background sync failed (0hkaa5): say so with its fix, after
	// the command's own output and before next.
	if sy, ok := reply["sync"].(Reply); ok {
		fmt.Fprintln(a.Out, "sync: "+sy["message"].(string))
		fmt.Fprintln(a.Out, "fix: "+sy["fix"].(string))
	}
	if msg, ok := reply["message"].(string); ok {
		fmt.Fprintln(a.Out, msg)
	} else if ids, ok := reply["renamed"].([]string); ok && len(ids) > 0 {
		if len(ids) > 10 {
			fmt.Fprintf(a.Out, "renamed: %s and %d more (files only, see --json)\n", strings.Join(ids[:10], ", "), len(ids)-10)
		} else {
			fmt.Fprintln(a.Out, "renamed: "+strings.Join(ids, ", "))
		}
	}
	fmt.Fprintln(a.Out, "next: "+reply["next"].(string))
	return 0
}

func shortHash(h string) string {
	if len(h) > 8 {
		return h[:8]
	}
	if h == "" {
		return "(no commit yet)"
	}
	return h
}

// childIndexFor memoises task.ChildIndex for the slice being rendered, so
// listing N tasks resolves parents once, not N times. Locked: an MCP server
// or parallel tests run commands from several goroutines.
var childMemo struct {
	sync.Mutex
	first *task.Task
	n     int
	idx   map[string][]task.Task
}

func childIndexFor(ts []task.Task) map[string][]task.Task {
	if len(ts) == 0 {
		return nil
	}
	childMemo.Lock()
	defer childMemo.Unlock()
	if childMemo.first != &ts[0] || childMemo.n != len(ts) {
		childMemo.first, childMemo.n, childMemo.idx = &ts[0], len(ts), task.ChildIndex(ts)
	}
	return childMemo.idx
}

// descriptionKey is the JSON key of a description section: purpose,
// principle, details (jnaspy; "details" as in hug task new --details).
func descriptionKey(section string) string {
	if section == "Relevant details" {
		return "details"
	}
	return strings.ToLower(section)
}

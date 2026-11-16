// Package cli is the shared non-interactive command surface for CLI and MCP.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/cache"
	"github.com/veridit/hug/huginn/internal/editor"
	"github.com/veridit/hug/huginn/internal/harness"
	"github.com/veridit/hug/huginn/internal/task"
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
  done                      hug task done k7m2qx
  check                     hug task check k7m2qx 1
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

var Commands = []string{"setup", "init", "new", "ls", "show", "edit", "move", "mv", "take", "done", "check", "promote", "note", "dep", "set", "drop", "next", "rebalance", "types", "lint", "sync", "import", "board", "tui", "mcp", "install", "uninstall", "upgrade", "attach", "detach", "help"}

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
	return exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/heads/"+OldTaskBranch).Run() == nil ||
		exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "--verify", "--quiet", "refs/remotes/origin/"+OldTaskBranch).Run() == nil
}

func (a *App) open(ctx context.Context) (*session, error) {
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	if check := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--show-toplevel"); check.Run() != nil {
		return nil, &OperationError{Problem: "hug task needs a Git checkout as its working directory", Fix: "cd /path/to/your/git/repo && hug task init"}
	}
	if root, e := gitbranch.MainRoot(cwd); e == nil && oldLayout(ctx, root) {
		return nil, &OperationError{Problem: "tasks are still on the old layout (branch huginn at .huginn/); hug now keeps them on branch hug/tasks at .hug/tasks/",
			Fix: "hug upgrade"}
	}
	// One board per repository: a linked worktree (git worktree add) uses the
	// main worktree's hug/tasks checkout, never a second one (065wwn).
	r, e := gitbranch.Open(cwd, TaskBranch, TaskDir)
	if e != nil {
		return nil, e
	}
	cmd := exec.CommandContext(ctx, "git", "-C", r.Dir, "rev-parse", "--git-common-dir")
	b, e := cmd.Output()
	if e != nil {
		return nil, e
	}
	gd := strings.TrimSpace(string(b))
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(r.Dir, gd)
	}
	c, e := cache.Open(r.Dir, filepath.Join(gd, "huginn", "cache.db"), gitCache{r})
	if e != nil {
		return nil, e
	}
	// One command reads the board once (44w1ev): List syncs, and the Store
	// keeps the result until it writes a task file itself.
	s := &task.Store{Root: r.Dir, Snapshot: true}
	s.Refresh = func() ([]task.Task, error) { return c.List(ctx) }
	if _, e = s.Tasks(); e != nil {
		c.Close()
		return nil, e
	}
	return &session{r, c, s}, nil
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
func harnessName() string {
	if id := strings.TrimSpace(os.Getenv("JCODE_SESSION_ID")); strings.HasPrefix(id, "session_") {
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
		t.Model = strings.TrimSpace(os.Getenv("HUG_MODEL"))
	}
	if t.Name == "" {
		t.Name = strings.TrimSpace(os.Getenv("HUG_USER"))
	}
	if t.Name == "" {
		t.Name = harnessName()
	}
	return t
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
		if slices.Contains([]string{"json", "jsonl", "wide", "help", "mine", "top", "dry-run", "all-instances", "all", "replace", "summary"}, key) {
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
var refCommands = []string{"show", "edit", "move", "mv", "take", "done", "drop", "check", "promote", "note", "dep", "set", "attach", "detach"}
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
		acceptance = append(acceptance, map[string]any{"text": item.Text, "checked": item.Checked})
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
	children := []Reply{}
	for _, c := range childIndexFor(ts)[strings.ToLower(t.ID)] {
		children = append(children, Reply{"id": c.ID, "title": c.Title, "status": c.Status})
	}
	out["children"] = children
	out["notes"] = ""
	if b, e := os.ReadFile(filepath.Join(root, filepath.FromSlash(t.Path))); e == nil {
		// Purpose, Principle and Relevant details follow the goal (jnaspy).
		d := task.Parse(t.Path, b)
		for _, sec := range task.DescriptionSections {
			out[descriptionKey(sec)] = d.SectionText(sec)
		}
		lines := strings.Split(string(b), "\n")
		active := false
		notes := []string{}
		for _, l := range lines {
			if strings.HasPrefix(l, "## ") {
				if active {
					break
				}
				active = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(l, "## ")), "Notes")
				continue
			}
			if active {
				notes = append(notes, l)
			}
		}
		out["notes"] = strings.TrimSpace(strings.Join(notes, "\n"))
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
func (a *App) Execute(ctx context.Context, args []string) (Reply, error) {
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
	allowed := map[string]string{"install": "all-instances jcode-home dry-run", "uninstall": "all-instances jcode-home dry-run", "new": "type priority goal purpose principle details accept dep group label top after", "ls": "status type priority label group mine jsonl epic as model", "board": "type priority label group mine all epic wide as model", "next": "type priority label group mine epic as model", "take": "as model branch worktree", "tui": "as model", "move": "before after top", "dep": "add remove", "set": "priority type label status title assignee parent", "import": "dry-run cite", "show": "file", "lint": "code summary all", "attach": "name note replace safe-reason content"}
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
	noArgs := []string{"lint", "ls", "next", "sync", "types", "init", "board", "setup", "tui", "upgrade"}
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
	s, e := a.open(ctx)
	if e != nil {
		return nil, e
	}
	defer s.c.Close()
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
		e = s.r.Commit(nil, "hug task init")
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
		return Reply{"task": taskJSON(d.Task, s.r.Dir, ts), "files": files, "next": task.NextHint(d.Task)}, nil
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
		if e = s.r.Commit(nil, "hug task attach "+r.Task.ID); e != nil {
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
		if e = f.count(0); e != nil {
			return nil, e
		}
		mode, code, e := lintFlags(f)
		if e != nil {
			return nil, e
		}
		fs, e := s.c.Lint(ctx)
		if e != nil {
			return nil, e
		}
		forbid, e := task.ForbidCite(s.r.Dir)
		if e != nil {
			return nil, e
		}
		more, e := s.s.LintFiles(forbid)
		if e != nil {
			return nil, e
		}
		fs = append(fs, more...)
		// Git branch state, not file content: recomputed on every lint (x1t1t1).
		if ts, e := s.s.Tasks(); e == nil {
			fs = append(fs, task.BranchFindings(a.Cwd, gitbranch.DefaultBranch(a.Cwd), ts)...)
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
		if e = s.r.Sync(); e != nil {
			return nil, e
		}
		s.s.Invalidate() // the pull may have changed task files
		return empty()
	case "new":
		if e = f.count(1); e != nil {
			return nil, e
		}
		r, e = s.s.New(task.NewOptions{Title: f.args[0], Goal: f.get("goal"), Purpose: f.get("purpose"), Principle: f.get("principle"), Details: f.get("details"), Type: f.get("type"), Priority: f.get("priority"), Group: f.get("group"), Acceptance: f.values["accept"], Deps: f.values["dep"], Labels: f.values["label"], Top: f.has("top"), After: f.get("after")})
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
		if e = f.count(1); e != nil {
			return nil, e
		}
		if cmd == "done" {
			r, e = s.s.Done(f.args[0])
		} else {
			r, e = s.s.Drop(f.args[0])
		}
	case "check":
		if e = f.count(2); e != nil {
			return nil, e
		}
		n, err := strconv.Atoi(f.args[1])
		if err != nil || n < 1 {
			return nil, usage("acceptance index must be a positive integer", "hug task show "+f.args[0])
		}
		r, e = s.s.Check(f.args[0], n)
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
			if e = s.r.Commit(nil, "hug task promote "+f.args[0]+" "+f.args[1]); e != nil {
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
		for _, field := range fields {
			if f.has(field) {
				count++
			}
		}
		if count == 0 {
			return nil, usage("set needs a field: --priority, --type, --label, --status, --title, --assignee, --parent", "hug task show "+f.args[0])
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
		ed := editor.Command()
		c := exec.CommandContext(ctx, ed[0], append(ed[1:], filepath.Join(s.r.Dir, d.Task.Path))...)
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
		if e = s.r.Commit(nil, "hug task edit "+d.Task.ID); e != nil {
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
		} else if e = s.r.Commit(nil, fmt.Sprintf("hug task rebalance (%d files)", len(r.Renamed))); e != nil {
			return nil, e
		}
		return Reply{"changed": r.Changed, "next": r.Next, "renamed": r.Renamed, "renames": r.Renames, "message": msg}, nil
	}
	if cmd != "next" && cmd != "take" {
		if e = s.r.Commit(nil, "hug task "+cmd+" "+r.Task.ID); e != nil {
			return nil, e
		}
	}
	reply, e := resultJSON(r, s)
	if e == nil && r.Changed {
		withGuide(reply, cmd, r.Task, s)
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
		if jsonMode {
			_ = json.NewEncoder(a.Out).Encode(Reply{"error": p, "fix": fix, "next": fix, "exit_code": code})
		} else {
			fmt.Fprintln(a.Err, p)
			fmt.Fprintln(a.Err, "fix: "+fix)
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
			writeBoard(a.Out, cols)
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
			mark := " "
			if m["checked"] == true {
				mark = "x"
			}
			fmt.Fprintf(a.Out, "%d. [%s] %s\n", i+1, mark, m["text"])
		}
		if t["notes"] != "" {
			fmt.Fprintf(a.Out, "notes: %s\n", t["notes"])
		}
		if ds, ok := t["dependency_details"].([]task.DependencyDetail); ok {
			for _, d := range ds {
				fmt.Fprintf(a.Out, "dependency: %s  %s  [%s]  %s\n", d.ID, d.Title, d.Status, d.Path)
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
	if fs, ok := reply["findings"].([]task.Finding); ok {
		sum, _ := reply["summary"].(LintSummary)
		writeFindings(a.Out, fs, sum)
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
	if ws, ok := reply["warnings"].([]string); ok {
		for _, v := range ws {
			fmt.Fprintln(a.Out, "warning: "+v)
		}
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
// listing N tasks resolves parents once, not N times.
var childMemo struct {
	first *task.Task
	n     int
	idx   map[string][]task.Task
}

func childIndexFor(ts []task.Task) map[string][]task.Task {
	if len(ts) == 0 {
		return nil
	}
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

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
  take                      hug task take k7m2qx
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
Identity: HUGINN_USER, otherwise git config user.name.
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
	if top, e := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--show-toplevel").Output(); e == nil {
		if root := strings.TrimSpace(string(top)); oldLayout(ctx, root) {
			return nil, &OperationError{Problem: "tasks are still on the old layout (branch huginn at .huginn/); hug now keeps them on branch hug/tasks at .hug/tasks/",
				Fix: "hug upgrade"}
		}
	}
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
	s := &task.Store{Root: r.Dir}
	s.Refresh = func() ([]task.Task, error) {
		if _, e := c.Sync(ctx); e != nil {
			return nil, e
		}
		return c.List(ctx)
	}
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
	c := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--show-toplevel")
	b, e := c.Output()
	if e != nil {
		return nil
	}
	if _, e = os.Lstat(filepath.Join(strings.TrimSpace(string(b)), TaskDir)); os.IsNotExist(e) {
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

func (a *App) identity(ctx context.Context) (string, error) {
	if u := strings.TrimSpace(os.Getenv("HUGINN_USER")); u != "" {
		return u, nil
	}
	cmd := exec.CommandContext(ctx, "git", "config", "user.name")
	cmd.Dir = a.Cwd
	b, e := cmd.Output()
	if e == nil && strings.TrimSpace(string(b)) != "" {
		return strings.TrimSpace(string(b)), nil
	}
	return "", usage("identity missing: set HUGINN_USER or git config user.name", `export HUGINN_USER="your-name"`)
}

type flags struct {
	values map[string][]string
	args   []string
}

func parse(args []string, allowed string) (flags, error) {
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
		if slices.Contains([]string{"json", "jsonl", "wide", "help", "mine", "top", "dry-run", "all-instances", "all", "replace"}, key) {
			if eq && val != "true" {
				return f, usage("--"+key+" is a boolean flag", "hug task help")
			}
			f.values[key] = []string{"true"}
			continue
		}
		if !eq {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
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
	allowed := map[string]string{"install": "all-instances jcode-home dry-run", "uninstall": "all-instances jcode-home dry-run", "new": "type priority goal purpose principle details accept dep group label top after", "ls": "status type priority label group mine jsonl epic", "board": "type priority label group mine all epic wide", "next": "type priority label group mine epic", "move": "before after top", "dep": "add remove", "set": "priority type label status title assignee parent", "import": "dry-run cite", "show": "file", "lint": "code", "attach": "name note replace safe-reason content"}
	f, e := parse(args, allowed[cmd])
	if e != nil {
		return nil, e
	}
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
		filter.Mine, e = a.identity(ctx)
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
		if c := f.get("code"); c != "" {
			kept := []task.Finding{}
			for _, x := range fs {
				if task.FindingCode(x.Problem) == c {
					kept = append(kept, x)
				}
			}
			fs = kept
		}
		if fs == nil {
			fs = []task.Finding{}
		}
		reply, e := empty()
		reply["findings"] = fs
		return reply, e
	case "sync":
		if e = f.count(0); e != nil {
			return nil, e
		}
		if e = s.r.Sync(); e != nil {
			return nil, e
		}
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
		r, e = a.take(ctx, s, f.args[0])
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
		if e = s.r.Commit(nil, "hug task edit "+d.Task.ID); e != nil {
			return nil, e
		}
		fs, err := s.c.Lint(ctx)
		if err != nil {
			return nil, err
		}
		reply, err := empty()
		if fs == nil {
			fs = []task.Finding{}
		}
		reply["findings"] = fs
		reply["next"] = "hug task show " + d.Task.ID
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
		for _, t := range ts {
			fmt.Fprintf(a.Out, "%-6s  %-7s  %-7s  %-*s  %s\n", t["id"], t["status"], t["type"], w, t["slug"], t["title"])
		}
	}
	if t, ok := reply["task"].(Reply); ok {
		fmt.Fprintf(a.Out, "%s  %s  %s  %s\n", t["slug"], t["title"], t["id"], t["status"])
		fmt.Fprintf(a.Out, "goal: %s\n", t["goal"])
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
		for _, v := range fs {
			fmt.Fprintln(a.Out, v.String())
		}
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

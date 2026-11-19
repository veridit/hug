package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/importer"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/huginn/internal/tui"
	"github.com/veridit/hug/internal/hugconf"
)

// Import and TUI hooks are deliberately thin: their packages own their behavior.
func (a *App) runImport(ctx context.Context, s *session, f flags) (Reply, error) {
	if len(f.args) >= 1 && f.args[0] == "github" {
		return a.runImportGitHub(ctx, s, f)
	}
	if len(f.args) >= 1 && f.args[0] == "files" {
		if e := f.count(2); e != nil {
			return nil, f.wrongCount("import files needs the folder of per-task folders")
		}
		dry := f.has("dry-run")
		r, e := importer.ImportFiles(f.args[1], f.get("cite"), s.s, dry)
		if e != nil {
			return nil, e
		}
		if dry {
			return Reply{"import": r, "next": "hug task import files " + f.args[1]}, nil
		}
		if e = s.commit("hug task import files " + f.args[1]); e != nil {
			return nil, e
		}
		return Reply{"import": r, "next": "hug task lint"}, nil
	}
	if e := f.count(2); e != nil {
		return nil, e
	}
	if f.args[0] != "backlog" {
		return nil, usage("unknown import format "+f.args[0]+"; valid: backlog, github, files; closest: backlog", "hug task import backlog "+f.args[1])
	}
	if f.has("cleanup") {
		return a.runImportCleanup(ctx, s, f, f.args[1])
	}
	// A source retired by --cleanup is never imported again (ggydj2).
	src := f.args[1]
	if rel, top, e := a.importSource(ctx, src); e == nil {
		if e := refuseRecordedImport(s.r.Dir, rel); e != nil {
			return nil, e
		}
		// DIR is relative to the command's folder, not the process's.
		src = filepath.Join(top, filepath.FromSlash(rel))
	}
	dry := f.has("dry-run")
	listing, listed := a.backlogListing(src)
	start := time.Now()
	report, e := importer.Run(ctx, src, s.s, importer.Options{DryRun: dry})
	var none *importer.NoTasksError
	if errors.As(e, &none) {
		return nil, usage(none.Error()+"; point at the backlog folder (holding tasks/) or the tasks folder itself", "hug task import backlog "+filepath.Join(f.args[1], "tasks"))
	}
	if e != nil {
		return nil, e
	}
	if dry {
		if listed {
			a.rememberBacklog(ctx, s.r.Common, src, listing, start, report)
		}
		return Reply{"import": report, "dry_run": true, "next": "hug task import backlog " + f.args[1]}, nil
	}
	if e = s.commit("hug task import backlog"); e != nil {
		return nil, e
	}
	// hug setup reads this instead of a dry run (3mavpt).
	if listed {
		a.rememberBacklog(ctx, s.r.Common, src, listing, start, importedReport(report))
	}
	s.s.Invalidate() // the import wrote task files
	if _, e = s.s.Tasks(); e != nil {
		return nil, e
	}
	return Reply{"import": report, "next": "once the import is complete, retire the source: hug task import backlog " + f.args[1] + " --cleanup --dry-run"}, nil
}
func (a *App) runTUI(ctx context.Context, s *session, f flags) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	if f.has("json") {
		return nil, usage("tui is interactive; use ls --json for structured output", "hug task ls --json")
	}
	// Taking from the TUI uses the same rule as hug task take: the model is
	// required (--model or HUG_MODEL), the name is optional (et85h1).
	who := a.whoTakes(f)
	who.At = a.takePlace(f) // the TUI takes too
	// A person at the keyboard: with no model given, TUI commits are theirs
	// (t709he; a CLI or MCP write never guesses human).
	if who.Model == "" {
		s.r.Trailers = []string{"Model: human"}
		// The TUI's t takes as the person at the keyboard (6ephqk check 6:
		// review_by = the TUI user, Model: human).
		who.Model = "human"
	}
	// The TUI lives while others write: it must read the files each time.
	s.s.Snapshot = false
	s.s.Invalidate()
	// Who "you" is (89y0b7, Jørgen 15:31): an agent is its own name (--as,
	// HUG_USER, the harness session), whatever git email it commits with;
	// a person is found by git email in hug.toml [people] and named by the
	// short name, else the email before the @. A model never stands in.
	agent := who.Name
	email := ""
	if agent == "" {
		if out, e := exec.CommandContext(ctx, "git", "-C", a.Cwd, "config", "user.email").Output(); e == nil {
			email = strings.TrimSpace(string(out))
		}
		people, _ := task.People(s.r.Dir)
		if p, ok := task.PersonByEmail(people, email); ok {
			who.Name = p.Short
		} else if email != "" {
			who.Name = task.EmailShort(email)
		}
	}
	// The TUI user's writes mark their unread answers read (3jtfw9).
	s.s.Actor = who.assignee()
	// The first frame is the board the cache last saw, read from the
	// database by Open, without a walk or git (vnyz8z); the sync runs after
	// it, behind the spinner. A first start (no cache) shows the spinner
	// and says it is reading the board.
	initial := &tui.Initial{Tasks: s.c.Listed()}
	if len(initial.Tasks) > 0 {
		initial.Findings = s.c.LintListed()
	}
	opts := tui.Options{User: agent, Email: email, Version: a.Version, MainBranch: gitbranch.DefaultBranch(a.Cwd), Take: func(ref string) (task.Result, error) { return a.take(ctx, s, ref, who) },
		// Every TUI write holds the board turn from changing a file to its
		// commit (95ahtk), so Commit runs inside it and does not take it
		// again (a second flock in one process would wait on itself).
		Turn: s.turn, Commit: s.commit,
		Initial: initial,
		// The refresh syncs the cache with the files; the reload after it
		// lists what the sync found.
		Refresh:  func(ctx context.Context) error { _, e := s.c.Sync(ctx); return e },
		Progress: s.c.Progress,
		// Findings come from the cache, which parsed each file once, as of
		// the listing the reload just made (44w1ev).
		Lint: func() ([]task.Finding, error) { return s.c.LintListed(), nil },
		// / search ranks by the cache's index, as hug task search (49n9kw).
		Search: func(query string) ([]string, error) { return s.c.Rank(ctx, query) },
		// r and a deliver as hug task request and answer do (gmzd87).
		Deliver: func(id, kind, by string) (string, error) {
			return a.Deliver(s.s, s.r.Dir, s.r.Root, id, kind, by, s.commit)
		}}
	// / and : histories (xfv9pw): per board, in the git dir beside the task
	// cache (one repository, every worktree, one history), never on the
	// task branch.
	opts.HistoryFile = filepath.Join(s.r.Common, "huginn", "tui-history.json")
	// The header names the project by this checkout's folder (0sw0dd):
	// a linked worktree reads as itself, hug-testing beside hug.
	opts.Checkout = s.r.Root
	// The : palette's keyless actions (05r3pg) run the command itself,
	// through Execute, the code the shell and MCP task_update run: it takes
	// the turn and commits on its own. The TUI's store then reads the files
	// again, and the cache syncs, so the board it draws is what was written.
	opts.Run = func(args []string) (string, error) {
		reply, e := a.Execute(ctx, args)
		s.s.Invalidate()
		_, _ = s.c.Sync(ctx)
		if e != nil {
			_, problem, fix := ErrorDetails(e)
			return "", fmt.Errorf("%s\nfix: %s", problem, fix)
		}
		return runSummary(args, reply), nil
	}
	// Not a terminal (a test's input): keys from In, no screen.
	if _, tty := a.In.(*os.File); !tty && a.In != nil {
		opts.Input, opts.Output, opts.NoReload = a.In, a.Out, true
	}
	e := tui.Run(ctx, s.s, opts)
	if e != nil {
		return nil, e
	}
	// Whatever the TUI left uncommitted, under the turn like any write: at
	// Seid this commit met an agent's hug task attach on index.lock
	// (2026-11-17 16:49:23, 95ahtk).
	release := s.turn()
	e = s.commit("hug task tui")
	release()
	if e != nil {
		return nil, e
	}
	return Reply{"next": "hug task next"}, nil
}

// runImportGitHub imports a repository's issues (open and closed) through
// the gh CLI: each issue becomes a task with alias GH-<number>. Re-running
// merges GitHub changes; edits made in huginn win, as with backlog imports.
func (a *App) runImportGitHub(ctx context.Context, s *session, f flags) (Reply, error) {
	if len(f.args) > 2 {
		return nil, f.wrongCount("import github takes at most OWNER/REPO")
	}
	repo := ""
	if len(f.args) == 2 {
		repo = f.args[1]
	}
	issues, e := importer.FetchIssues(ctx, a.Cwd, repo)
	if e != nil {
		return nil, &OperationError{Problem: e.Error(), Fix: "gh auth status, then hug task import github OWNER/REPO --dry-run"}
	}
	stage, e := os.MkdirTemp("", "huginn-github-*")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(stage)
	if e = importer.WriteIssues(stage, issues); e != nil {
		return nil, e
	}
	dry := f.has("dry-run")
	report, e := importer.Run(ctx, stage, s.s, importer.Options{DryRun: dry, Origin: "github"})
	var none *importer.NoTasksError
	if errors.As(e, &none) {
		return Reply{"import": report, "issues": 0, "next": "hug task ls"}, nil
	}
	if e != nil {
		return nil, e
	}
	again := "hug task import github"
	if repo != "" {
		again += " " + repo
	}
	if dry {
		return Reply{"import": report, "issues": len(issues), "dry_run": true, "next": again}, nil
	}
	if e = s.commit("hug task import github"); e != nil {
		return nil, e
	}
	s.s.Invalidate() // the import wrote task files
	if _, e = s.s.Tasks(); e != nil {
		return nil, e
	}
	return Reply{"import": report, "issues": len(issues), "next": "hug task ls"}, nil
}

// runUpgrade does every mechanical migration (asr4t2): the old huginn
// layout becomes hug's (branch, worktree, settings file), every task file is
// tidied and says state: (6ephqk step 4), old waits on a person get
// asked, hook variables get HUG_ names, and a legacy memory file is
// named with its command. It says per part what it changed, or "nothing to
// do", then points at hug setup for choices. It previews by default
// (Jørgen 2026-11-17): nothing changes, each change reads "would: ...", the
// diff comes along, and with changes pending it ends fix: hug upgrade
// --apply and exits UpgradePending. --apply does the work. Safe to run
// again.
func (a *App) runUpgrade(ctx context.Context, f flags) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	dry := !f.has("apply")
	// This machine's files (hook and MCP configs, the memory store) change
	// only with --apply --machine (f5vx4x, crab 04:04): a board's upgrade
	// never reaches them. 2026-11-18 04:53 an upgrade in a scratch clone
	// rewrote six live jcode configs.
	machine := f.has("machine")
	if machine && dry {
		return nil, usage("--machine goes with --apply: hug upgrade shows this machine's changes already", "hug upgrade --apply --machine")
	}
	machineDry := !machine
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	var parts, machineParts []string
	upgradeDiff := ""
	pending, machinePending := false, false
	part := func(name string, did []string) {
		onMachine := name == "variables" || name == "memory" || name == "names"
		// memory only names a command; it is never pending here. A skipped
		// file is a line for a person, not a change upgrade will make.
		changes := 0
		for _, d := range did {
			if !strings.HasPrefix(d, "skipped ") {
				changes++
			}
		}
		if changes > 0 && name != "memory" {
			if onMachine {
				machinePending = machinePending || machineDry
			} else {
				pending = pending || dry
			}
		}
		out := &parts
		if onMachine {
			out = &machineParts
		}
		if len(did) == 0 {
			*out = append(*out, name+": nothing to do")
			return
		}
		for _, d := range did {
			*out = append(*out, name+": "+d)
		}
	}
	would := func(s string) string {
		if dry {
			return "would: " + s
		}
		return s
	}
	wouldMachine := func(s string) string {
		if machineDry {
			return "would: " + s
		}
		return s
	}
	// Layout: the board branch, its worktree and the settings file.
	var layout []string
	if dry {
		if top, e := gitbranch.MainRoot(cwd); e == nil {
			if _, e := os.Stat(filepath.Join(top, OldTaskDir)); e == nil {
				layout = append(layout, would(OldTaskDir+"/ moves to "+TaskDir+"/, branch "+OldTaskBranch+" becomes "+TaskBranch))
			}
		}
	} else {
		m, e := gitbranch.Upgrade(cwd, OldTaskBranch, OldTaskDir, TaskBranch, TaskDir)
		if e != nil {
			return nil, e
		}
		if m.LocalBranch {
			layout = append(layout, "branch "+OldTaskBranch+" renamed to "+TaskBranch)
		}
		if m.Worktree {
			layout = append(layout, OldTaskDir+"/ moved to "+TaskDir+"/")
		}
		if m.RemoteBranch {
			layout = append(layout, "remote branch renamed")
		}
	}
	// Settings file: huginn.toml becomes hug.toml (committed on the task branch).
	if top, e := gitbranch.MainRoot(cwd); e == nil {
		dir := filepath.Join(top, TaskDir)
		if _, e := os.Stat(filepath.Join(dir, "huginn.toml")); e == nil {
			if _, e := os.Stat(filepath.Join(dir, task.ConfigName)); os.IsNotExist(e) {
				if !dry {
					if out, e := exec.CommandContext(ctx, "git", "-C", dir, "mv", "huginn.toml", task.ConfigName).CombinedOutput(); e != nil {
						return nil, &OperationError{Problem: "rename huginn.toml: " + strings.TrimSpace(string(out)), Fix: "git -C " + dir + " mv huginn.toml " + task.ConfigName}
					}
					if out, e := exec.CommandContext(ctx, "git", "-C", dir, "commit", "-q", "-m", "hug upgrade: huginn.toml -> "+task.ConfigName).CombinedOutput(); e != nil {
						return nil, &OperationError{Problem: "commit rename: " + strings.TrimSpace(string(out)), Fix: "git -C " + dir + " status"}
					}
				}
				layout = append(layout, would("huginn.toml renamed to "+task.ConfigName))
			}
		}
	}
	memoryLayout, err := upgradeMemoryLayout(cwd, dry)
	if err != nil {
		return nil, err
	}
	layout = append(layout, memoryLayout...)
	part("layout", layout)
	// The board's task files, only when the board opens: a repository
	// without hug/tasks has nothing to migrate.
	var tasks []string
	boardDir := ""
	if s, e := a.open(ctx); e == nil {
		defer s.c.Close()
		boardDir = s.r.Dir
		// The background pass takes Exclusive then Turn. Use that same
		// order and fail closed: no rebase or other writer may move HEAD
		// while upgrade is planning or installing its replay.
		locks, e := s.r.Background()
		if e != nil {
			return nil, e
		}
		release, e := locks.Exclusive()
		if e != nil {
			return nil, e
		}
		defer release()
		turn, e := locks.Turn()
		if e != nil {
			return nil, e
		}
		defer turn()
		s.s.Invalidate()
		// Catch-up first (w40cmk check 9): the remote moved to this
		// layout and this clone has commits from before. They are
		// replayed onto the remote one by one, so every step below works
		// on the board as everyone has it.
		lines, e := a.catchUp(s, dry)
		if e != nil {
			return nil, e
		}
		caughtUp := len(lines) > 0
		if caughtUp {
			part("catch-up", []string{strings.Join(lines, "\n    ")})
		}
		diffBefore := len(upgradeDiff)
		// An older hug's upgrade would move the board back (dx9vw2).
		if board, known, newer := a.newerBoard(s.r.Dir); newer {
			return nil, boardNewer(board, known)
		}
		// state: (6ephqk step 4): every status: key becomes state:, first,
		// so every later step reads the new key.
		changed, lines, e := s.s.MigrateState(dry)
		if e == nil && len(changed) > 0 && !dry {
			e = s.r.Commit(changed, fmt.Sprintf("hug upgrade: status: -> state: in %d tasks", len(changed)))
		}
		if e != nil {
			return nil, e
		}
		if len(changed) > 0 {
			tasks = append(tasks, would(fmt.Sprintf("%d tasks moved to state: (was status:)", len(changed))))
			upgradeDiff += strings.Join(lines, "\n") + "\n"
		}
		// Section order (jnaspy): Goal, Purpose, Principle, Relevant
		// details, Depends on, Acceptance, others, Notes, History, with each
		// section's text unchanged; backlog leftovers go.
		changed, diff, e := s.s.Tidy(dry)
		if e == nil && len(changed) > 0 && !dry {
			e = s.r.Commit(changed, fmt.Sprintf("hug upgrade: backlog leftovers and section order in %d tasks", len(changed)))
		}
		if e != nil {
			return nil, e
		}
		if len(changed) > 0 {
			tasks = append(tasks, would(fmt.Sprintf("%d tasks tidied: backlog leftovers removed, sections put in order (no section text changed)", len(changed))))
			upgradeDiff += diff
		}
		// Questions to a person (41hhgn): a doing/waiting with waiting_for
		// (or, before 89y0b7, a waiting_on "NAME: what" with NAME in
		// hug.toml [people]) becomes doing/asked.
		people, _ := task.People(s.r.Dir)
		moved, wlines, left, e := s.s.MoveToAsked(people, dry)
		asked := len(moved) - len(left)
		if e == nil && len(moved) > 0 && !dry {
			msg := fmt.Sprintf("hug upgrade: %d questions to a person moved from waiting to asked", asked)
			if len(left) > 0 {
				msg += fmt.Sprintf(", %d waits left waiting (no question)", len(left))
			}
			e = s.r.Commit(moved, msg)
		}
		if e != nil {
			return nil, e
		}
		if asked > 0 {
			ids := make([]string, 0, len(wlines))
			for _, l := range wlines {
				id, _, _ := strings.Cut(l, ":")
				ids = append(ids, id)
			}
			tasks = append(tasks, would(fmt.Sprintf("%d questions to a person moved from waiting to asked (%s)", asked, strings.Join(ids, ", "))))
			upgradeDiff += strings.Join(wlines, "\n") + "\n"
		}
		// A wait that named a person but asks nothing (a park) stays a
		// wait (41hhgn check 13): each is listed, with the ask that makes
		// it a question if it was one.
		if len(left) > 0 {
			noun := "waits"
			if len(left) == 1 {
				noun = "wait"
			}
			line := fmt.Sprintf("%d %s on a person left waiting (it asks nothing: no '?', no check named, no request such as approve); waiting_for goes:", len(left), noun)
			for _, l := range left {
				line += fmt.Sprintf(" %s waiting_for %s: %s (if it is a question: hug task ask %s --to %s \"...?\");", l.ID, l.For, strconv.Quote(l.What), l.ID, l.For)
			}
			tasks = append(tasks, would(strings.TrimSuffix(line, ";")))
		}
		// Layout 3 (w40cmk): every task to C/ID.md, its rank and group in
		// its front matter, its files folder to C/ID/, in one commit.
		from, _, _ := a.newerBoard(s.r.Dir)
		plan, e := s.s.PlanLayout3()
		if e != nil {
			return nil, e
		}
		if len(plan.Moves) > 0 {
			if !dry {
				paths, e := s.s.ApplyLayout3(plan)
				// The layout goes in the same commit: no commit has C/ID.md
				// files on a board that says layout 2 (check 1: one commit).
				if e == nil {
					e = writeLayout(s.r.Dir, task.Layout)
					paths = append(paths, task.ConfigName)
				}
				if e == nil {
					e = s.r.Commit(paths, fmt.Sprintf("hug upgrade: %d tasks to C/ID.md, rank: and group: in the file (layout %d)", len(plan.Moves), task.Layout))
				}
				if e != nil {
					return nil, e
				}
			}
			line := fmt.Sprintf("%d tasks move to C/ID.md (the path never changes again), rank: and group: set in each file", len(plan.Moves))
			if plan.Files > 0 {
				line += fmt.Sprintf(", %d files folders to C/ID/", plan.Files)
			}
			if len(plan.Groups) > 0 {
				line += fmt.Sprintf(", group folders %s become group: values", strings.Join(plan.Groups, ", "))
			}
			tasks = append(tasks, would(line))
			var b strings.Builder
			for _, m := range plan.Moves {
				fmt.Fprintf(&b, "%s: %s -> %s (rank %q, group %q)\n", m.ID, m.From, m.To, m.Rank, m.Group)
			}
			upgradeDiff += b.String()
		}
		// Last, once the task files are moved: the board says its layout,
		// so an older hug stops before writing it (dx9vw2).
		if board, known, _ := a.newerBoard(s.r.Dir); board < known {
			if !dry {
				if e := writeLayout(s.r.Dir, known); e != nil {
					return nil, e
				}
				if e := s.r.Commit([]string{task.ConfigName}, fmt.Sprintf("hug upgrade: board layout %d -> %d", board, known)); e != nil {
					return nil, e
				}
			}
			tasks = append(tasks, would(fmt.Sprintf("board layout %d -> %d in %s (an older hug then refuses to write it)", board, known, task.ConfigName)))
		} else if len(plan.Moves) > 0 && !dry {
			tasks = append(tasks, fmt.Sprintf("board layout %d -> %d in %s, in the same commit (an older hug then refuses to write it)", from, known, task.ConfigName))
		}
		if caughtUp && dry {
			// The steps above read this clone's board as it is; after the
			// catch-up it is the remote's, so what they would do here is
			// not what --apply does. --apply runs them after the catch-up.
			tasks = []string{would("run after the catch-up, on the remote's board (hug upgrade shows them then)")}
			upgradeDiff = upgradeDiff[:diffBefore]
		}
		part("tasks", tasks)
	} else {
		parts = append(parts, "tasks: no board here (hug task init makes one)")
	}
	// Variables (m1c28r): old names hug wrote into hook commands and its MCP
	// server entries become HUG_ names, each changed file backed up.
	var vars []string
	backedUp := map[string]bool{}
	home, homeErr := a.homeDir()
	if homeErr == nil {
		files, e := renameEnv(envFiles(home, a.getenv("JCODE_HOME")), machineDry)
		if e != nil {
			return nil, &OperationError{Problem: "rename variables in " + strings.Join(files, ", ") + ": " + e.Error(), Fix: "hug upgrade"}
		}
		for _, f := range files {
			backedUp[f] = !machineDry
			if machineDry {
				vars = append(vars, wouldMachine("variables renamed to HUG_* in "+f))
			} else {
				vars = append(vars, "variables renamed to HUG_* in "+f+" (backup "+filepath.Base(f)+".bak-hug-upgrade)")
			}
		}
	}
	part("variables", vars)
	// Names (ynh84t): hooks and MCP entries an older hug wrote under the
	// muninn and huginn names run and name hug. In a preview the variables
	// step changed nothing, so names reads the same files it would.
	var names []string
	if homeErr == nil {
		moves, e := moveNames(nameFiles(home, a.getenv("JCODE_HOME")), hugconf.HugPath(), machineDry, backedUp)
		if e != nil {
			return nil, &OperationError{Problem: "move hook and MCP names: " + e.Error(), Fix: "hug upgrade"}
		}
		moved := false
		for _, m := range moves {
			switch {
			case m.Foreign:
				names = append(names, "skipped "+m.File+`: its "hug" MCP entry is not hug's, so the old one stays (fix: rename that entry, then hug upgrade --apply --machine)`)
			case machineDry:
				names = append(names, wouldMachine("hooks and MCP entries run hug in "+m.File))
				moved = true
			default:
				names = append(names, "hooks and MCP entries run hug in "+m.File+" (backup "+filepath.Base(m.File)+".bak-hug-upgrade)")
				moved = true
			}
		}
		if moved {
			// The old names are links to this hug: what is not moved, and a
			// server still running under them, keeps working.
			names = append(names, "the muninn and huginn links stay, so anything not moved keeps working; an MCP server started under the old name runs until reloaded (jcode: mcp action=reload server=hug; Claude, Codex: restart the app)")
		}
	}
	part("names", names)
	// Memory: a legacy .muninn/memories.jsonl moves off the code branch with
	// its own command, which talks to the remote; upgrade names it.
	var mem []string
	if top, e := gitbranch.MainRoot(cwd); e == nil {
		if _, e := os.Stat(filepath.Join(top, ".muninn", "memories.jsonl")); e == nil {
			mem = append(mem, "a legacy .muninn/memories.jsonl is on the code branch: hug mem sync --migrate moves it (hug mem sync --migrate --dry-run shows how)")
		}
	}
	// Twins older syncs made of this machine's own memories (5sc5rw):
	// hug mem upgrade folds them, previewing like every other part.
	foldPending := false
	if a.MemUpgrade != nil {
		args := []string{}
		if !machineDry {
			args = append(args, "--apply")
		}
		code, out := a.MemUpgrade(args)
		switch code {
		case 0, UpgradePending:
			for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
				l = strings.TrimPrefix(l, "memory: ")
				if l == "" || l == "nothing to do" || strings.HasPrefix(l, "fix: ") {
					continue
				}
				mem = append(mem, l)
			}
			foldPending = code == UpgradePending
		default:
			return nil, &OperationError{Problem: "hug mem upgrade failed: " + strings.TrimSpace(out), Fix: "hug mem upgrade"}
		}
	}
	part("memory", mem)
	machinePending = machinePending || foldPending
	reply := Reply{"upgrade": append(append([]string(nil), parts...), machineParts...), "board": parts, "machine": machineParts, "board_dir": boardDir, "next": "hug setup"}
	switch {
	case dry && pending:
		reply["pending"] = true
		reply["fix"] = "hug upgrade --apply"
		reply["next"] = "hug upgrade --apply"
		reply["exit_code"] = UpgradePending
		if machinePending {
			reply["then"] = "hug upgrade --apply --machine"
		}
	case machinePending:
		// The board is done (or had nothing); this machine's part waits for
		// its own step: exit 5, since upgrade has changes pending (crab
		// 04:04), so a script never reads half an upgrade as done.
		reply["pending"] = true
		reply["fix"] = "hug upgrade --apply --machine"
		reply["next"] = "hug upgrade --apply --machine"
		reply["exit_code"] = UpgradePending
	}
	if upgradeDiff != "" {
		reply["diff"] = upgradeDiff
	}
	return reply, nil
}

// writeLayout sets layout = n in the board's hug.toml (dx9vw2).
func writeLayout(dir string, n int) error {
	path := filepath.Join(dir, task.ConfigName)
	b, e := os.ReadFile(path)
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	return os.WriteFile(path, []byte(task.WithLayout(string(b), n)), 0o644)
}

// runSummary is one line for the TUI's message after a palette command
// (05r3pg): the command as typed, and whether it changed anything; the
// reply's next step when it names one.
func runSummary(args []string, reply Reply) string {
	line := "hug task " + strings.Join(args, " ")
	if len(line) > 60 {
		line = line[:57] + "..."
	}
	said := "done: " + line
	if changed, ok := reply["changed"].(bool); ok && !changed {
		said = "nothing to change: " + line
	}
	if next, ok := reply["next"].(string); ok && next != "" {
		said += " · next: " + next
	}
	return said
}

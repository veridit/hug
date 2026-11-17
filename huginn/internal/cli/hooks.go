package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/importer"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/huginn/internal/tui"
)

// Import and TUI hooks are deliberately thin: their packages own their behavior.
func (a *App) runImport(ctx context.Context, s *session, f flags) (Reply, error) {
	if len(f.args) >= 1 && f.args[0] == "github" {
		return a.runImportGitHub(ctx, s, f)
	}
	if len(f.args) >= 1 && f.args[0] == "files" {
		if e := f.count(2); e != nil {
			return nil, usage("import files needs the folder of per-task folders", "hug task import files DIR [--cite PREFIX] [--dry-run]")
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
	report, e := importer.Run(ctx, src, s.s, importer.Options{DryRun: dry})
	var none *importer.NoTasksError
	if errors.As(e, &none) {
		return nil, usage(none.Error()+"; point at the backlog folder (holding tasks/) or the tasks folder itself", "hug task import backlog "+filepath.Join(f.args[1], "tasks"))
	}
	if e != nil {
		return nil, e
	}
	if dry {
		return Reply{"import": report, "dry_run": true, "next": "hug task import backlog " + f.args[1]}, nil
	}
	if e = s.commit("hug task import backlog"); e != nil {
		return nil, e
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
		Search: func(query string) ([]string, error) { return s.c.Rank(ctx, query) }}
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
		return nil, usage("too many arguments", "hug task import github [OWNER/REPO] [--dry-run]")
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

// runUpgrade moves the old huginn layout to hug's: branch huginn becomes
// hug/tasks, worktree .huginn/ moves to .hug/tasks/, and the remote branch is
// renamed. History is unchanged. Safe to run again.
func (a *App) runUpgrade(ctx context.Context, f flags) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	cwd := a.Cwd
	if cwd == "" {
		cwd = "."
	}
	m, e := gitbranch.Upgrade(cwd, OldTaskBranch, OldTaskDir, TaskBranch, TaskDir)
	if e != nil {
		return nil, e
	}
	did := []string{}
	upgradeDiff := ""
	if m.LocalBranch {
		did = append(did, "branch "+OldTaskBranch+" renamed to "+TaskBranch)
	}
	if m.Worktree {
		did = append(did, OldTaskDir+"/ moved to "+TaskDir+"/")
	}
	if m.RemoteBranch {
		did = append(did, "remote branch renamed")
	}
	// Settings file: huginn.toml becomes hug.toml (committed on the task branch).
	if top, e := gitbranch.MainRoot(cwd); e == nil {
		dir := filepath.Join(top, TaskDir)
		if _, e := os.Stat(filepath.Join(dir, "huginn.toml")); e == nil {
			if _, e := os.Stat(filepath.Join(dir, task.ConfigName)); os.IsNotExist(e) {
				if out, e := exec.CommandContext(ctx, "git", "-C", dir, "mv", "huginn.toml", task.ConfigName).CombinedOutput(); e != nil {
					return nil, &OperationError{Problem: "rename huginn.toml: " + strings.TrimSpace(string(out)), Fix: "git -C " + dir + " mv huginn.toml " + task.ConfigName}
				}
				if out, e := exec.CommandContext(ctx, "git", "-C", dir, "commit", "-q", "-m", "hug upgrade: huginn.toml -> "+task.ConfigName).CombinedOutput(); e != nil {
					return nil, &OperationError{Problem: "commit rename: " + strings.TrimSpace(string(out)), Fix: "git -C " + dir + " status"}
				}
				did = append(did, "huginn.toml renamed to "+task.ConfigName)
			}
		}
	}
	// Section order (jnaspy): every task file in Goal, Purpose, Principle,
	// Relevant details, Depends on, Acceptance, others, Notes, History, with
	// each section's text unchanged. Only when the board opens: a repository
	// without hug/tasks has nothing to reorder.
	if s, e := a.open(ctx); e == nil {
		changed, diff, e := s.s.Tidy(false)
		if e == nil && len(changed) > 0 {
			e = s.r.Commit(changed, fmt.Sprintf("hug upgrade: backlog leftovers and section order in %d tasks", len(changed)))
		}
		// The cache stays open for the waits below (89y0b7: closing it here
		// made the move fail with "sql: database is closed").
		defer s.c.Close()
		if e != nil {
			return nil, e
		}
		if len(changed) > 0 {
			did = append(did, fmt.Sprintf("%d tasks tidied: backlog leftovers removed, sections put in order (no section text changed)", len(changed)))
			upgradeDiff = diff
		}
		// Waits on a person (89y0b7): an old waiting_on "NAME: what"
		// with NAME in hug.toml [people] gets waiting_for NAME.
		if e == nil {
			people, _ := task.People(s.r.Dir)
			moved, lines, e2 := s.s.MoveWaitingFor(people, false)
			if e2 == nil && len(moved) > 0 {
				e2 = s.r.Commit(moved, fmt.Sprintf("hug upgrade: waiting_for in %d tasks", len(moved)))
			}
			if e2 != nil {
				return nil, e2
			}
			if len(moved) > 0 {
				did = append(did, fmt.Sprintf("%d waits on a person moved to waiting_for", len(moved)))
				upgradeDiff += strings.Join(lines, "\n") + "\n"
			}
		}
	}
	// Variables (m1c28r): old names hug wrote into hook commands and its MCP
	// server entries become HUG_ names, each changed file backed up.
	if home, e := os.UserHomeDir(); e == nil {
		files, e := renameEnv(envFiles(home), false)
		if e != nil {
			return nil, &OperationError{Problem: "rename variables in " + strings.Join(files, ", ") + ": " + e.Error(), Fix: "hug upgrade"}
		}
		for _, f := range files {
			did = append(did, "variables renamed to HUG_* in "+f+" (backup "+filepath.Base(f)+".bak-hug-upgrade)")
		}
	}
	if len(did) == 0 {
		did = append(did, "already on the hug layout")
	}
	reply := Reply{"upgrade": did, "next": "hug setup"}
	if upgradeDiff != "" {
		reply["diff"] = upgradeDiff
	}
	return reply, nil
}

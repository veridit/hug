package cli

import (
	"context"
	"errors"
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
	dry := f.has("dry-run")
	report, e := importer.Run(ctx, f.args[1], s.s, importer.Options{DryRun: dry})
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
	if _, e = s.s.Tasks(); e != nil {
		return nil, e
	}
	return Reply{"import": report, "next": "hug task ls"}, nil
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
	e := tui.Run(ctx, s.s, tui.Options{User: who.assignee(), Version: a.Version, Take: func(ref string) (task.Result, error) { return a.take(ctx, s, ref, who) }, Commit: func(msg string) error { defer s.turn()(); return s.commit(msg) }})
	if e != nil {
		return nil, e
	}
	if e = s.commit("hug task tui"); e != nil {
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
	if len(did) == 0 {
		did = append(did, "already on the hug layout")
	}
	return Reply{"upgrade": did, "next": "hug setup"}, nil
}

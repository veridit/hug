package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/veridit/hug/huginn/internal/importer"
	"github.com/veridit/hug/huginn/internal/task"
	"github.com/veridit/hug/huginn/internal/tui"
)

// Import and TUI hooks are deliberately thin: their packages own their behavior.
func (a *App) runImport(ctx context.Context, s *session, f flags) (Reply, error) {
	if len(f.args) >= 1 && f.args[0] == "github" {
		return a.runImportGitHub(ctx, s, f)
	}
	if e := f.count(2); e != nil {
		return nil, e
	}
	if f.args[0] != "backlog" {
		return nil, usage("unknown import format "+f.args[0]+"; valid: backlog, github; closest: backlog", "huginn import backlog "+f.args[1])
	}
	dry := f.has("dry-run")
	report, e := importer.Run(ctx, f.args[1], s.s, importer.Options{DryRun: dry})
	var none *importer.NoTasksError
	if errors.As(e, &none) {
		return nil, usage(none.Error()+"; point at the backlog folder (holding tasks/) or the tasks folder itself", "huginn import backlog "+filepath.Join(f.args[1], "tasks"))
	}
	if e != nil {
		return nil, e
	}
	if dry {
		return Reply{"import": report, "dry_run": true, "next": "huginn import backlog " + f.args[1]}, nil
	}
	if e = s.r.Commit(nil, "huginn import backlog"); e != nil {
		return nil, e
	}
	if _, e = s.s.Tasks(); e != nil {
		return nil, e
	}
	return Reply{"import": report, "next": "huginn ls"}, nil
}
func (a *App) runTUI(ctx context.Context, s *session, f flags) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	if f.has("json") {
		return nil, usage("tui is interactive; use ls --json for structured output", "huginn ls --json")
	}
	user, _ := a.identity(ctx)
	e := tui.Run(ctx, s.s, tui.Options{User: user, Take: func(ref string) (task.Result, error) { return a.take(ctx, s, ref) }, Commit: func(msg string) error { return s.r.Commit(nil, msg) }})
	if e != nil {
		return nil, e
	}
	if e = s.r.Commit(nil, "huginn tui"); e != nil {
		return nil, e
	}
	return Reply{"next": "huginn next"}, nil
}

// runImportGitHub imports a repository's issues (open and closed) through
// the gh CLI: each issue becomes a task with alias GH-<number>. Re-running
// merges GitHub changes; edits made in huginn win, as with backlog imports.
func (a *App) runImportGitHub(ctx context.Context, s *session, f flags) (Reply, error) {
	if len(f.args) > 2 {
		return nil, usage("too many arguments", "huginn import github [OWNER/REPO] [--dry-run]")
	}
	repo := ""
	if len(f.args) == 2 {
		repo = f.args[1]
	}
	issues, e := importer.FetchIssues(ctx, a.Cwd, repo)
	if e != nil {
		return nil, &OperationError{Problem: e.Error(), Fix: "gh auth status, then huginn import github OWNER/REPO --dry-run"}
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
		return Reply{"import": report, "issues": 0, "next": "huginn ls"}, nil
	}
	if e != nil {
		return nil, e
	}
	again := "huginn import github"
	if repo != "" {
		again += " " + repo
	}
	if dry {
		return Reply{"import": report, "issues": len(issues), "dry_run": true, "next": again}, nil
	}
	if e = s.r.Commit(nil, "huginn import github"); e != nil {
		return nil, e
	}
	if _, e = s.s.Tasks(); e != nil {
		return nil, e
	}
	return Reply{"import": report, "issues": len(issues), "next": "huginn ls"}, nil
}

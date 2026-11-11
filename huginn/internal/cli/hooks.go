package cli

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/veridit/huginn/internal/importer"
	"github.com/veridit/huginn/internal/task"
	"github.com/veridit/huginn/internal/tui"
)

// Import and TUI hooks are deliberately thin: their packages own their behavior.
func (a *App) runImport(ctx context.Context, s *session, f flags) (Reply, error) {
	if e := f.count(2); e != nil {
		return nil, e
	}
	if f.args[0] != "backlog" {
		return nil, usage("unknown import format "+f.args[0]+"; valid: backlog; closest: backlog", "huginn import backlog "+f.args[1])
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

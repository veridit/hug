package cli

import (
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/task"
)

// take is the one push-lock path, shared by CLI, MCP and the TUI callback.
// who must name a model (et85h1); without one nothing is pulled or locked.
func (a *App) take(ctx context.Context, s *session, ref string, who taker) (task.Result, error) {
	if who.Model == "" {
		id := ref
		if d, err := s.s.Get(ref); err == nil {
			id = d.Task.ID
		}
		fix := "hug task take " + id + " --model MODEL"
		if who.Name != "" {
			fix = "hug task take " + id + " --as " + who.Name + " --model MODEL"
		}
		return task.Result{}, usage("no model given: say which model takes "+id+" (claude-opus-5-5, gpt-6.1-sol, ...; a person: --model human), with --model or HUG_MODEL", fix)
	}
	// Publish unrelated local history before gitbranch's lost-race rollback.
	// Otherwise an unpublished new task could disappear along with a losing take.
	if s.r.Remote != "" {
		ahead := exec.CommandContext(ctx, "git", "-C", s.r.Dir, "rev-list", "--count", "refs/remotes/"+s.r.Remote+"/"+s.r.Branch+"..HEAD")
		count, err := ahead.Output()
		if err != nil || strings.TrimSpace(string(count)) != "0" {
			if err = s.r.Sync(); err != nil {
				return task.Result{}, err
			}
		}
	}
	var result task.Result
	e := s.r.UpdateMessage("hug task take "+ref, func(_ *gitbranch.Repository) error {
		// Update has pulled already. Read authoritative files without invoking Git
		// inside its callback, then refresh the disposable cache after the lock.
		local := *s.s
		local.Refresh = nil
		var err error
		result, err = local.TakeAt(ref, who.Name, who.Model, who.At)
		return err
	})
	var raced *gitbranch.ErrRaced
	if errors.As(e, &raced) {
		d, err := s.s.Get(ref)
		if err != nil {
			return task.Result{}, err
		}
		return task.Result{}, &OperationError{Problem: "taken by " + strings.Join(d.Task.Assignee, ", ") + " (" + d.Task.ID + ")", Fix: "hug task next"}
	}
	if e == nil {
		_, e = s.s.Tasks()
	}
	return result, e
}

// placeOf is the branch and worktree folder of dir (x1t1t1). Outside git, or
// with a detached HEAD, it is empty and nothing is recorded.
func placeOf(dir string) task.Place {
	if dir == "" {
		return task.Place{}
	}
	p, err := gitbranch.Where(dir)
	if err != nil || p.Branch == "" {
		return task.Place{}
	}
	return task.Place{Branch: p.Branch, Worktree: p.Worktree}
}

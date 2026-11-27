package cli

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

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
	// No background pass fetches or pushes meanwhile, and no write commits
	// between the pull and the push (0hkaa5).
	defer s.exclusive()()
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
		if who.With != "" {
			result, err = local.TakeAtWith(ref, who.With, who.Name, who.Model, who.At)
		} else {
			result, err = local.TakeAt(ref, who.Name, who.Model, who.At)
		}
		return err
	})
	var raced *gitbranch.ErrRaced
	if errors.As(e, &raced) {
		s.s.Invalidate()
		d, err := s.s.Get(ref)
		if err != nil {
			return task.Result{}, err
		}
		return task.Result{}, &OperationError{Problem: "taken by " + strings.Join(d.Task.Assignee, ", ") + " (" + d.Task.ID + ")", Fix: "hug task next"}
	}
	// The pull and the take changed files behind the Store's back.
	s.s.Invalidate()
	if e == nil {
		_, e = s.s.Tasks()
	}
	// Taken by an owner whose session is gone (9ahvgt check 3, mizaru on
	// txjkrm): the fix takes it over, not "hug task next". The same test
	// as lint's stale-owner; the take itself is still refused.
	var te *task.Error
	if errors.As(e, &te) && te.Kind == "taken" && who.Name != "" {
		if d, err := s.s.Get(te.ID); err == nil {
			people, _ := task.People(s.r.Dir)
			if fs := task.OwnerFindings(s.liveness(a), []task.Task{d.Task}, time.Now(), people...); len(fs) > 0 {
				takeAgain := "hug task take " + d.Task.ID + " --as " + who.Name + " --model " + who.Model
				return task.Result{}, &OperationError{
					Problem: "taken by " + te.Value + " (" + d.Task.ID + "), but " + strings.TrimPrefix(fs[0].Problem, "owner "+te.Value+" of "+d.Task.ID+" "),
					Fix:     "hug task set " + d.Task.ID + " --assignee " + who.Name + " && " + takeAgain,
				}
			}
		}
	}
	// Read the ticket first: its Goal, details and checks are what the
	// todos are written from (tzc9hb).
	if e == nil && result.Changed {
		result.Next = "hug task show " + result.Task.ID
		// A task with no acceptance check has no work items yet (erv680
		// check 1): writing them is the step, before anything else.
		if st := task.StepOf(result.Task, task.StepConfig{}); st.Kind == task.StepChecks {
			result.Next = st.Next
		}
	}
	// The board says who is working (v3k6cp): a name other than the
	// harness session's is taken as given, and said, so a coordinator's
	// "you are humpback" does not quietly put the wrong owner on the board.
	if e == nil && who.Name != "" {
		if h := harnessName(a.getenv("JCODE_SESSION_ID")); h != "" && !strings.EqualFold(h, who.Name) {
			result.Warnings = append(result.Warnings, "assignee "+who.Name+" is not this session ("+h+"); if you are "+h+": hug task set "+result.Task.ID+" --assignee "+h)
		}
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

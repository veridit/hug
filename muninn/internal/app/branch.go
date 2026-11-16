package app

import (
	"github.com/veridit/hug/huginn/gitbranch"
)

// WorkBranch is the branch to record on a memory written from cwd (x1t1t1):
// the checked-out branch when cwd is in a linked worktree, or on a branch
// other than the main worktree's. Work on the main checkout's own branch, a
// detached HEAD or outside git records nothing.
func WorkBranch(cwd string) string {
	if cwd == "" {
		return ""
	}
	here, err := gitbranch.Where(cwd)
	if err != nil || here.Branch == "" {
		return ""
	}
	if here.Linked {
		return here.Branch
	}
	// The main checkout switched to a feature branch is branch work too.
	if d := gitbranch.DefaultBranch(here.Main); d != "" && here.Branch != d {
		return here.Branch
	}
	return ""
}

// workBranch is WorkBranch for the folder this binding was opened for.
func (b Binding) workBranch() string {
	if b.context == nil {
		return ""
	}
	return WorkBranch(b.context.cwd)
}

// workRepo is the main checkout of the repository the binding's folder is
// in, "" outside git: where a memory's task or branch condition is checked
// (ez50rb).
func (b Binding) workRepo() string {
	if b.context == nil || b.context.cwd == "" {
		return ""
	}
	here, err := gitbranch.Where(b.context.cwd)
	if err != nil {
		return ""
	}
	return here.Main
}

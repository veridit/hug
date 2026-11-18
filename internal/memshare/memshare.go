// Package memshare answers one question for every path that could put a
// repository's project memories on its remote: does this repository opt in
// to sharing them? The memory sync (muninn project.SyncBranch) and the
// layout upgrade (hug upgrade, gitbranch.Upgrade) both ask it, so a public
// repository never publishes memories by a route that forgot to check
// (34r1ft, wxsyc5).
//
// The setting (34r1ft check 1, Jørgen 2026-11-18 18:18, "Clearly A") is
// committed with the repository's tasks, so every clone, fresh ones too,
// follows the repository rather than the machine:
//
//	# hug.toml on branch hug/tasks
//	[memory]
//	share = true
//
// Absent, false or unreadable means local: nothing is pushed. hug task
// memory share writes it; hug task memory local removes it.
package memshare

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Fix is the command that opts a repository in.
const Fix = "hug task memory share"

// ErrLocal: this repository keeps its project memories on this machine.
var ErrLocal = errors.New("this repository keeps project memories local: hug.toml [memory] share = true is not committed on hug/tasks, so nothing was written to a branch or pushed (a private repository opts in; a public one leaves it local)\nfix: " + Fix)

// KeptNote is what hug upgrade says when it renamed the memory branch here
// only and left the remote's old branch as it was.
const KeptNote = "remote muninn-notes kept; this repo keeps memories local"

// The task branch and its settings file (huginn/internal/cli TaskBranch,
// task.ConfigName).
const (
	taskBranch = "hug/tasks"
	configName = "hug.toml"
)

// Shares reports whether the repository at dir (any directory in any of
// its worktrees) lets its project memories reach a branch on its remote:
// hug.toml [memory] share = true as committed on hug/tasks, or on
// origin/hug/tasks before this clone has its own. Absent means no; outside
// a git repository, no. A value other than true or false is an error.
func Shares(dir string) (bool, error) {
	if exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run() != nil {
		return false, nil
	}
	for _, ref := range []string{"refs/heads/" + taskBranch, "refs/remotes/origin/" + taskBranch} {
		out, err := exec.Command("git", "-C", dir, "cat-file", "blob", ref+":"+configName).Output()
		if err != nil {
			continue // no such branch, or no hug.toml on it
		}
		return Parse(out)
	}
	return false, nil
}

// Parse reads [memory] share from hug.toml's content.
func Parse(content []byte) (bool, error) {
	section := ""
	share := false
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || section != "memory" || strings.TrimSpace(key) != "share" {
			continue
		}
		if n := strings.Index(value, "#"); n >= 0 {
			value = value[:n]
		}
		switch strings.TrimSpace(value) {
		case "true":
			share = true
		case "false":
			share = false
		default:
			return false, fmt.Errorf("hug.toml [memory] share = %s: want true or false; memories stay local (hug task memory local writes the default)\nfix: %s", strings.TrimSpace(value), Fix)
		}
	}
	return share, nil
}

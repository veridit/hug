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

// Fix is the command that opts a repository in. It is for a person: hug
// never prints it as a fix: or next: line on a local repository, since an
// agent runs those as given and in a public repository that would publish
// (crab review of e2582a0f).
const Fix = "hug task memory share"

// LocalStatus is what hug mem sync says on a repository that keeps its
// memories local: the intended state, not an error.
const LocalStatus = "project memories stay on this machine (hug.toml [memory] share is not set)"

// PersonHint is the one sentence naming the opt-in, for a person reading
// hug task memory; never a fix: or next: line.
const PersonHint = "a private repository can share them with: " + Fix

// ErrLocal: this repository keeps its project memories on this machine.
// Callers that print it print LocalStatus and succeed.
var ErrLocal = errors.New(LocalStatus)

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
	out, ok := Config(dir)
	if !ok {
		return false, nil
	}
	return Parse(out)
}

// Config returns hug.toml as committed on hug/tasks (or origin/hug/tasks
// before this clone has its own), and false when there is none.
func Config(dir string) ([]byte, bool) {
	if exec.Command("git", "-C", dir, "rev-parse", "--git-dir").Run() != nil {
		return nil, false
	}
	for _, ref := range []string{"refs/heads/" + taskBranch, "refs/remotes/origin/" + taskBranch} {
		out, err := exec.Command("git", "-C", dir, "cat-file", "blob", ref+":"+configName).Output()
		if err != nil {
			continue // no such branch, or no hug.toml on it
		}
		return out, true
	}
	return nil, false
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
			return false, fmt.Errorf("hug.toml [memory] share = %s: want true or false; memories stay local\nfix: hug task memory local", strings.TrimSpace(value))
		}
	}
	return share, nil
}

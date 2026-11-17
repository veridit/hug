package cond

import (
	"os/exec"
	"strings"
	"sync"
	"time"
)

// LoadAverage is the 5-minute load average of this machine; tests replace
// it to inject a load.
var LoadAverage = systemLoad

// MemoFor is how long a Local state keeps a git answer: one command asks
// once, and a long-lived TUI sees a merge within this time.
var MemoFor = 10 * time.Second

type answer struct {
	ok, known bool
	why       string
	at        time.Time
}

// Local is the State of the git checkout at dir and this machine: the load
// average, branches merged into the main branch (mainBranch names it, asked
// once and only when a branch is asked about; checked locally and on
// origin), and tags. Git is asked lazily, at most
// once per question within MemoFor, and never over the network: a tag or
// merge other clones made shows after the next fetch (hug fetches in the
// background on reads).
func Local(dir string, mainBranch func() string) State {
	var mainOnce sync.Once
	var main string
	var mu sync.Mutex
	memo := map[string]answer{}
	ask := func(key string, f func() answer) answer {
		mu.Lock()
		defer mu.Unlock()
		if a, ok := memo[key]; ok && time.Since(a.at) < MemoFor {
			return a
		}
		a := f()
		a.at = time.Now()
		memo[key] = a
		return a
	}
	git := func(args ...string) (string, bool) {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Env = append(c.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
		out, err := c.Output()
		return strings.TrimSpace(string(out)), err == nil
	}
	tip := func(ref string) string {
		out, ok := git("rev-parse", "--verify", "--quiet", ref+"^{commit}")
		if !ok {
			return ""
		}
		return out
	}
	return State{
		Load: LoadAverage,
		Branch: func(name, at string) (bool, string, bool) {
			a := ask("branch\x00"+name+"\x00"+at, func() answer {
				mainOnce.Do(func() {
					if mainBranch != nil {
						main = mainBranch()
					}
				})
				if main == "" {
					return answer{why: "no main branch here"}
				}
				if name == main {
					return answer{why: name + " is the main branch"}
				}
				// The recorded tip, when this clone has it (a wait written in
				// another clone may name a commit not fetched here), else
				// the branch as it is now.
				commit := ""
				if at != "" {
					commit = tip(at)
				}
				if commit == "" {
					if commit = tip("refs/heads/" + name); commit == "" {
						commit = tip("refs/remotes/origin/" + name)
					}
				}
				if commit == "" {
					return answer{why: "no branch " + name + " in this clone"}
				}
				var mains []string
				for _, ref := range []string{"refs/heads/" + main, "refs/remotes/origin/" + main} {
					if tip(ref) != "" {
						mains = append(mains, ref)
					}
				}
				if len(mains) == 0 {
					return answer{why: "no branch " + main + " in this clone"}
				}
				for _, m := range mains {
					if _, ok := git("merge-base", "--is-ancestor", commit, m); ok {
						return answer{ok: true, known: true, why: "branch " + name + " is merged into " + main}
					}
				}
				return answer{known: true, why: "branch " + name + " is not merged into " + main}
			})
			return a.ok, a.why, a.known
		},
		Tag: func(name string) (bool, bool) {
			a := ask("tag\x00"+name, func() answer {
				if _, ok := git("rev-parse", "--git-dir"); !ok {
					return answer{}
				}
				return answer{ok: tip("refs/tags/"+name) != "", known: true}
			})
			return a.ok, a.known
		},
	}
}

// Tip is the commit a branch points at in the checkout at dir (locally, else
// on origin), or empty: recorded with a branch condition when it is written.
func Tip(dir, name string) string {
	for _, ref := range []string{"refs/heads/" + name, "refs/remotes/origin/" + name} {
		c := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
		c.Env = append(c.Environ(), "GIT_OPTIONAL_LOCKS=0")
		if out, err := c.Output(); err == nil {
			if s := strings.TrimSpace(string(out)); s != "" {
				return s[:min(12, len(s))]
			}
		}
	}
	return ""
}

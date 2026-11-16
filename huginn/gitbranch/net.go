package gitbranch

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Network calls never hang (0hkaa5, Jørgen 2026-11-16: "graceful detection
// of network failure, never hang"). A git command that talks to the remote
// runs non-interactively and has a deadline:
//   - no prompts: GIT_TERMINAL_PROMPT=0 (always), no askpass helper, Git
//     Credential Manager never interactive, ssh BatchMode=yes, so a missing
//     key or password fails at once instead of waiting for a person;
//   - ssh gives up connecting after ConnectTimeout seconds;
//   - http gives up when the transfer stalls below 1 KB/s for 15 s;
//   - the whole command is killed after NetTimeout.
//
// A user's own ssh command (GIT_SSH_COMMAND, GIT_SSH or core.sshCommand) is
// kept: hug then adds only the deadline, never options it may not accept.
var (
	NetTimeout     = 60 * time.Second
	ConnectTimeout = 10
)

// netVerbs are the git commands that contact a remote.
var netVerbs = map[string]bool{"ls-remote": true, "fetch": true, "push": true, "pull": true, "clone": true}

// isNet reports whether args (after any -c options) contact a remote.
func isNet(args []string) bool {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" {
			i++
			continue
		}
		return netVerbs[args[i]]
	}
	return false
}

// netArgs prepends the stall limits to a network command's arguments.
func netArgs(args []string) []string {
	return append([]string{"-c", "http.lowSpeedLimit=1000", "-c", "http.lowSpeedTime=15"}, args...)
}

// netEnv adds the no-prompt settings to a network command's environment.
func netEnv(dir string, env []string) []string {
	env = append(env, "GIT_ASKPASS=", "SSH_ASKPASS=", "SSH_ASKPASS_REQUIRE=never", "GCM_INTERACTIVE=never")
	if os.Getenv("GIT_SSH_COMMAND") != "" || os.Getenv("GIT_SSH") != "" {
		return env
	}
	// GIT_SSH_COMMAND would override the user's core.sshCommand: look first.
	if out, e := exec.Command("git", "-C", dir, "config", "--get", "core.sshCommand").Output(); e == nil && strings.TrimSpace(string(out)) != "" {
		return env
	}
	return append(env, "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout="+strconv.Itoa(ConnectTimeout))
}

// netTimeout explains a network command killed at its deadline.
func netTimeout(args []string) string {
	return "the remote did not answer within " + NetTimeout.String() + " (git " + strings.Join(args, " ") + "), so hug stopped waiting instead of hanging; local commits are kept and pushed when the remote answers"
}

// ErrNetTimeout is the cause of an Error from a network command killed at
// NetTimeout: callers keep their local commits and report them unpushed.
var ErrNetTimeout = errors.New("remote did not answer in time")

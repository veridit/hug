// Package tmuxtest names, finds and reaps the tmux servers tests start
// (wacc33).
//
// Every tmux server a hug test starts has the -L label Label(name):
// "huginn-NAME-PID", PID being the test process that owns it. A test
// process that dies hard (SIGKILL, a cancelled CI job) can leave its
// server behind. Reap ends exactly those: a huginn- label whose owner PID
// no longer runs, or whose parent is init/launchd (pid 1). A server whose
// test still runs is never touched, nor is any server without the prefix.
package tmuxtest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// Prefix begins the -L label of every tmux server a hug test starts.
const Prefix = "huginn-"

// Label is the -L label for a test's tmux server: Prefix, name, and this
// process's pid, so the next run can tell when its owner is gone.
func Label(name string) string { return fmt.Sprintf("%s%s-%d", Prefix, name, os.Getpid()) }

// Proc is a running tmux server: its pid, its parent and its -L label.
type Proc struct {
	PID, PPID int
	Label     string
}

// A tmux server's command line is "tmux -D -L label ..." where tmux keeps
// its argv (macOS), or "tmux: server (/tmp/tmux-UID/label)" where it sets
// its process title (Linux, proc.c setproctitle).
var serverArgs = regexp.MustCompile(`^(?:\S*/)?tmux(?:: server \(\S*/([^/)]+)\)| -D -L (\S+))`)

// Servers lists the tmux servers on the machine.
func Servers() ([]Proc, error) {
	out, err := exec.Command("ps", "-A", "-o", "pid=", "-o", "ppid=", "-o", "args=").Output()
	if err != nil {
		return nil, err
	}
	return parse(string(out)), nil
}

func parse(ps string) []Proc {
	var procs []Proc
	for _, line := range strings.Split(ps, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		m := serverArgs.FindStringSubmatch(strings.Join(f[2:], " "))
		if m == nil {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		procs = append(procs, Proc{pid, ppid, m[1] + m[2]})
	}
	return procs
}

// Alive says whether server pid with this label still runs.
func Alive(pid int, label string) (bool, error) {
	procs, err := Servers()
	if err != nil {
		return false, err
	}
	for _, p := range procs {
		if p.PID == pid && p.Label == label {
			return true, nil
		}
	}
	return false, nil
}

// Owner is the test process pid at the end of a Label, or 0.
func Owner(label string) int {
	i := strings.LastIndexByte(label, '-')
	if !strings.HasPrefix(label, Prefix) || i < 0 {
		return 0
	}
	pid, err := strconv.Atoi(label[i+1:])
	if err != nil || pid <= 1 {
		return 0
	}
	return pid
}

// running says whether process pid exists (EPERM: it does, as another user's).
func running(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

// Orphans are the test servers left by a run that died: a Prefix label,
// and its owner gone or its parent init/launchd.
func Orphans(procs []Proc, running func(int) bool) []Proc {
	var out []Proc
	for _, p := range procs {
		if !strings.HasPrefix(p.Label, Prefix) {
			continue
		}
		if owner := Owner(p.Label); p.PPID == 1 || (owner != 0 && !running(owner)) {
			out = append(out, p)
		}
	}
	return out
}

// Reap sends SIGTERM (what tmux kill-server does) to every orphaned test
// server and returns the ones it signalled.
func Reap() ([]Proc, error) {
	procs, err := Servers()
	if err != nil {
		return nil, err
	}
	var reaped []Proc
	for _, p := range Orphans(procs, running) {
		proc, err := os.FindProcess(p.PID)
		if err == nil && proc.Signal(syscall.SIGTERM) == nil {
			reaped = append(reaped, p)
		}
	}
	return reaped, nil
}

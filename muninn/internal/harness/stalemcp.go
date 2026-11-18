package harness

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// z77pem: a hug mcp server started before v0.7.54 (mbdfzs) never reloads
// itself: it serves every call with the hug it started from, and its old
// rules fail in ways the agent cannot read (Statbus 2026-11-18:
// memory_remember -> "cannot modify graph_state because it is a view").
// hug cannot change an old binary, so the new hug says which servers are
// stale and how their app restarts them. It never stops them: they belong
// to another app.

// ReloadSince is the first hug whose mcp server reloads itself (mbdfzs).
const ReloadSince = "0.7.54"

// StaleMCP is one running hug mcp server too old to reload itself.
type StaleMCP struct {
	PID     int
	Version string // from its Cellar path
	Exe     string // the file it runs
	App     string // who started it: "jcode HOME", "Claude", or the parent's name
	Parents []int  // the app process serving it (its parent, or above a wrapper)
}

// MCPScan is what one look at the machine found: the stale servers, and
// this process with its ancestors (the hook runs under its session's app).
type MCPScan struct {
	Stale []StaleMCP
	Self  []int
}

// Own splits the stale servers into the one(s) serving this session and the
// rest (crab, z77pem review). A jcode session's server is the one of its
// jcode home; with no home (Claude Code, Codex) it is a server whose serving
// app is an ancestor of this hook. None found: own is empty, and the agent
// hears nothing it cannot act on.
func (sc MCPScan) Own(jcodeHome string) (own, others []StaleMCP) {
	self := map[int]bool{}
	for _, p := range sc.Self {
		if p > 1 {
			self[p] = true
		}
	}
	for _, s := range sc.Stale {
		mine := false
		if jcodeHome != "" {
			h, ok := strings.CutPrefix(s.App, "jcode ")
			mine = ok && filepath.Clean(h) == filepath.Clean(jcodeHome)
		} else {
			for _, p := range s.Parents {
				mine = mine || self[p]
			}
		}
		if mine {
			own = append(own, s)
		} else {
			others = append(others, s)
		}
	}
	return own, others
}

// Proc is one process as ps lists it.
type Proc struct {
	PID, PPID int
	Args      string
}

// cellarVersion finds the version in a Homebrew Cellar path of hug.
var cellarVersion = regexp.MustCompile(`/Cellar/(?:hug|huginn|muninn)/(\d+)\.(\d+)\.(\d+)[^/]*/`)

// ScanMCP lists the hug mcp servers on this machine whose binary is a
// Homebrew Cellar hug older than ReloadSince. A path with no version (a
// build, or a removed file macOS no longer names) is not judged: a server
// from v0.7.54 on reloads itself whatever happened to its file. macOS and
// Linux; elsewhere, or when ps fails, it finds none.
func ScanMCP(ctx context.Context) MCPScan {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return MCPScan{}
	}
	out, err := exec.CommandContext(ctx, "ps", "-A", "-o", "pid=", "-o", "ppid=", "-o", "args=").Output()
	if err != nil {
		return MCPScan{}
	}
	procs := ParsePS(out)
	sc := MCPScan{Self: ancestors(procs, os.Getpid(), 64)}
	servers := MCPServers(procs, os.Getpid())
	if len(servers) > 0 {
		sc.Stale = JudgeMCP(servers, procs, executables(ctx, servers))
	}
	return sc
}

// ancestors is pid and up to n of its ancestors, nearest first.
func ancestors(procs []Proc, pid, n int) []int {
	parent := map[int]int{}
	for _, p := range procs {
		parent[p.PID] = p.PPID
	}
	out := []int{pid}
	for i := 0; i < n; i++ {
		pp, ok := parent[pid]
		if !ok || pp == pid || pp <= 0 {
			break
		}
		out = append(out, pp)
		pid = pp
	}
	return out
}

// ParsePS reads `ps -o pid= -o ppid= -o args=` output.
func ParsePS(out []byte) []Proc {
	var ps []Proc
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		pid, e1 := strconv.Atoi(f[0])
		ppid, e2 := strconv.Atoi(f[1])
		if e1 != nil || e2 != nil {
			continue
		}
		ps = append(ps, Proc{PID: pid, PPID: ppid, Args: strings.Join(f[2:], " ")})
	}
	return ps
}

// MCPServers picks the processes that are hug mcp (or huginn/muninn mcp, the
// old names), never self.
func MCPServers(procs []Proc, self int) []Proc {
	var out []Proc
	for _, p := range procs {
		f := strings.Fields(p.Args)
		if p.PID == self || len(f) < 2 || f[1] != "mcp" {
			continue
		}
		switch filepath.Base(f[0]) {
		case "hug", "huginn", "muninn":
			out = append(out, p)
		}
	}
	return out
}

// JudgeMCP keeps the servers whose executable is a Cellar hug older than
// ReloadSince. exe maps a pid to the file it runs.
func JudgeMCP(servers, procs []Proc, exe map[int]string) []StaleMCP {
	parent := map[int]Proc{}
	for _, p := range procs {
		parent[p.PID] = p
	}
	var out []StaleMCP
	for _, s := range servers {
		// Linux names a removed file "PATH (deleted)"; its version still counts.
		path := strings.TrimSuffix(exe[s.PID], " (deleted)")
		m := cellarVersion.FindStringSubmatch(path)
		if m == nil {
			continue
		}
		v := m[1] + "." + m[2] + "." + m[3]
		if !versionBefore(v, ReloadSince) {
			continue // it reloads itself at its next call
		}
		out = append(out, StaleMCP{PID: s.PID, Version: v, Exe: path, App: appOf(s, parent[s.PPID]), Parents: servingApp(s, parent)})
	}
	return out
}

// servingApp is the process that runs server s for its session: its parent,
// and through a wrapper (sh -c, Claude's disclaimer) the one above it.
func servingApp(s Proc, parent map[int]Proc) []int {
	p := parent[s.PPID]
	out := []int{s.PPID}
	if f := strings.Fields(p.Args); len(f) > 0 {
		switch filepath.Base(f[0]) {
		case "sh", "bash", "zsh", "dash", "env", "disclaimer":
			out = append(out, p.PPID)
		}
	}
	return out
}

func versionBefore(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x < y
		}
	}
	return false
}

// appOf names who started server s: a jcode home (from --jcode-home, else
// the parent's builds path), Claude, or the parent's file name.
func appOf(s, parent Proc) string {
	f := strings.Fields(s.Args)
	for i, a := range f {
		if a == "--jcode-home" && i+1 < len(f) {
			return "jcode " + f[i+1]
		}
	}
	pa := parent.Args
	if i := strings.Index(pa, "/builds/"); i > 0 && strings.Contains(pa, "jcode") {
		return "jcode " + strings.Fields(pa[:i])[len(strings.Fields(pa[:i]))-1]
	}
	if strings.Contains(pa, "Claude.app") {
		return "Claude"
	}
	if pf := strings.Fields(pa); len(pf) > 0 {
		return filepath.Base(pf[0])
	}
	return ""
}

// executables maps each server's pid to the file it runs: /proc on Linux,
// one lsof call for all of them on macOS.
func executables(ctx context.Context, servers []Proc) map[int]string {
	exe := map[int]string{}
	if runtime.GOOS == "linux" {
		for _, s := range servers {
			if p, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", s.PID)); err == nil {
				exe[s.PID] = p
			}
		}
		return exe
	}
	pids := make([]string, len(servers))
	for i, s := range servers {
		pids[i] = strconv.Itoa(s.PID)
	}
	out, _ := exec.CommandContext(ctx, "lsof", "-a", "-p", strings.Join(pids, ","), "-d", "txt", "-Fpn").Output()
	return ParseLsofTxt(out)
}

// ParseLsofTxt reads `lsof -d txt -Fpn`: the first txt name after each p
// line is the process's executable.
func ParseLsofTxt(out []byte) map[int]string {
	exe := map[int]string{}
	pid := 0
	for _, line := range bytes.Split(out, []byte("\n")) {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(string(line[1:]))
		case 'n':
			if _, ok := exe[pid]; !ok && pid != 0 {
				exe[pid] = string(line[1:])
			}
		}
	}
	return exe
}

// OwnStaleMessage is what the agent reads when the hug mcp serving its own
// session is too old to reload itself: one line, and the fix it can run.
// "" when none is.
func OwnStaleMessage(own []StaleMCP) string {
	if len(own) == 0 {
		return ""
	}
	s := own[0]
	line := fmt.Sprintf("hug: your hug mcp server (pid %d) runs hug %s, older than v%s: it cannot reload itself, so its calls run old rules and can fail (e.g. memory_remember: cannot modify graph_state because it is a view)", s.PID, s.Version, ReloadSince)
	fix := "fix: mcp action=reload server=hug"
	if !strings.HasPrefix(s.App, "jcode ") {
		fix = "fix: restart hug's MCP server in " + s.App + " (quit and reopen it)"
		if s.App == "" {
			fix = "fix: restart this app's hug MCP server, or the app"
		}
	}
	return line + "\n" + fix
}

// StaleList is the stale servers in one line for a person (hook.log, hug
// mem stats): "pid 3175 hug 0.7.2 in jcode /x; pid 77273 hug 0.7.43 in
// Claude". "" for none.
func StaleList(ss []StaleMCP) string {
	var each []string
	for _, s := range ss {
		where := ""
		if s.App != "" {
			where = " in " + s.App
		}
		each = append(each, fmt.Sprintf("pid %d hug %s%s", s.PID, s.Version, where))
	}
	return strings.Join(each, "; ")
}

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// hug version --running (rvb3j5) lists the hug processes running on this
// machine and the hug each one runs, so no one needs lsof to see a server
// left on an old hug (2026-11-25: five hug mcp on 0.7.108 after 0.7.112 was
// installed). It reads ps, and lsof on macOS or /proc on Linux; it never
// signals or stops a process.

// runningProc is one hug process.
type runningProc struct {
	PID     int
	Runs    string // mcp, tui, task ls, mem serve, ...
	Started time.Time
	Exe     string // the file it runs, as the system names it
	Gone    bool   // that file was removed or replaced since it started
	Version string // "" when nothing says
	Args    string
}

// runningVersion finds the version in a Homebrew Cellar path of hug.
var runningVersion = regexp.MustCompile(`/Cellar/(?:hug|huginn|muninn)/([^/]+)/`)

const runningUnsupportedFix = "fix: on this system, look for hug processes in the task manager (or: tasklist | findstr hug)"

// versionRunning prints the running hug processes for goos: exit 0, 5 when
// one runs a hug older than this one, 1 when it cannot list them.
func versionRunning(w io.Writer, goos string) int {
	if goos != "darwin" && goos != "linux" {
		fmt.Fprintf(w, "hug version --running: listing running hug processes is not supported on %s\n%s\n", goos, runningUnsupportedFix)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	procs, err := listRunning(ctx, goos)
	if err != nil {
		fmt.Fprintf(w, "hug version --running: cannot list processes: %v\nfix: ps -A -o pid,lstart,args | grep hug\n", err)
		return 1
	}
	return printRunning(w, procs, Version)
}

// printRunning writes the table and the fix for any older hug.
func printRunning(w io.Writer, procs []runningProc, self string) int {
	if len(procs) == 0 {
		fmt.Fprintln(w, "no other hug process is running")
		return 0
	}
	_, selfOK := parseVersion(self)
	fmt.Fprintf(w, "%d other hug processes are running:\n", len(procs))
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "PID\tRUNS\tVERSION\tSTARTED\tFILE\t")
	older := map[string][]int{}
	for _, p := range procs {
		v, note := p.Version, ""
		if v == "" {
			v = "?"
		}
		if selfOK && versionOlder(p.Version, self) {
			note = "older than this hug (" + self + ")"
			older[p.Runs] = append(older[p.Runs], p.PID)
		} else if p.Version == "" {
			note = "version unknown"
		}
		exe := p.Exe
		if exe == "" {
			exe = "?"
		}
		if p.Gone {
			exe += " (removed or replaced since it started)"
		}
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n", p.PID, p.Runs, v, p.Started.Format("2006-01-02 15:04:05"), exe, note)
	}
	tw.Flush()
	if !selfOK {
		fmt.Fprintf(w, "this hug is %s, a build with no release version: versions are not compared\n", self)
		return 0
	}
	if len(older) == 0 {
		return 0
	}
	kinds := make([]string, 0, len(older))
	for k := range older {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Fprintf(w, "%s: pid %s run an older hug\n", k, joinPIDs(older[k]))
	}
	if _, ok := older["mcp"]; ok {
		fmt.Fprintln(w, "fix: a hug mcp from this release on reloads the installed hug by itself within a minute; an older one does not: restart it in its app (jcode: mcp action=reload server=hug; Claude, Codex: quit and reopen the app)")
	} else {
		fmt.Fprintln(w, "fix: quit and start them again (hug tui: R reloads when it shows a newer hug)")
	}
	return 5
}

func joinPIDs(ps []int) string {
	s := make([]string, len(ps))
	for i, p := range ps {
		s[i] = strconv.Itoa(p)
	}
	return strings.Join(s, ", ")
}

// parseVersion reads the X.Y.Z at the start of v ("0.7.114", "v0.7.2",
// "0.0.1-old").
func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// versionOlder says a is a release before b. Either unreadable: false.
func versionOlder(a, b string) bool {
	x, ok1 := parseVersion(a)
	y, ok2 := parseVersion(b)
	if !ok1 || !ok2 {
		return false
	}
	for i := range x {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return false
}

// psLine is one line of ps -o pid= -o lstart= -o args=.
type psLine struct {
	PID     int
	Started time.Time
	Args    []string
}

// parsePSStart reads `ps -A -o pid= -o lstart= -o args=` under LC_ALL=C:
// lstart is five fields, "Wed Nov 18 19:55:22 2026", in local time.
func parsePSStart(out []byte) []psLine {
	var ls []psLine
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 7 {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		if err != nil {
			continue
		}
		t, err := time.ParseInLocation("Mon Jan 2 15:04:05 2006", strings.Join(f[1:6], " "), time.Local)
		if err != nil {
			continue
		}
		ls = append(ls, psLine{PID: pid, Started: t, Args: f[6:]})
	}
	return ls
}

// hugRuns names what a hug command line runs ("" when it is not hug):
// mcp, tui, task ls, mem serve. The old part names count as their part.
func hugRuns(args []string) string {
	if len(args) == 0 {
		return ""
	}
	var part string
	switch strings.TrimSuffix(filepath.Base(args[0]), ".exe") {
	case "hug":
	case "huginn":
		part = "task"
	case "muninn":
		part = "mem"
	default:
		return ""
	}
	var words []string
	if part != "" {
		words = append(words, part)
	}
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "-") || len(words) == 2 {
			break
		}
		switch a {
		case "tasks", "huginn":
			a = "task"
		case "memory", "muninn":
			a = "mem"
		}
		words = append(words, a)
		if len(words) == 1 && a != "task" && a != "mem" {
			break // hug mcp, hug tui, hug version: one word says it
		}
	}
	if len(words) == 0 {
		return "hug"
	}
	return strings.Join(words, " ")
}

// pickHug keeps the hug processes of ps, never self.
func pickHug(ls []psLine, self int) []runningProc {
	var out []runningProc
	for _, l := range ls {
		if l.PID == self {
			continue
		}
		if runs := hugRuns(l.Args); runs != "" {
			out = append(out, runningProc{PID: l.PID, Runs: runs, Started: l.Started, Args: strings.Join(l.Args, " ")})
		}
	}
	return out
}

// listRunning reads the machine's hug processes and what each one runs.
func listRunning(ctx context.Context, goos string) ([]runningProc, error) {
	cmd := exec.CommandContext(ctx, "ps", "-A", "-o", "pid=", "-o", "lstart=", "-o", "args=")
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	procs := pickHug(parsePSStart(out), os.Getpid())
	exes := runningExes(ctx, goos, procs)
	asked := map[string]string{}
	for i := range procs {
		p := &procs[i]
		e := exes[p.PID]
		p.Exe, p.Gone = e.path, e.gone
		if m := runningVersion.FindStringSubmatch(p.Exe); m != nil {
			p.Version = m[1]
			continue
		}
		if p.Gone || !filepath.IsAbs(p.Exe) {
			continue // the file now there is another hug: it cannot say
		}
		v, ok := asked[p.Exe]
		if !ok {
			v = askVersion(ctx, p.Exe)
			asked[p.Exe] = v
		}
		p.Version = v
	}
	return procs, nil
}

// askVersion runs path version: the file a process runs, unchanged since it
// started, says its own version.
func askVersion(ctx context.Context, path string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		return ""
	}
	v, ok := strings.CutPrefix(strings.TrimSpace(string(out)), "hug ")
	if !ok || strings.ContainsAny(v, " \n") {
		return ""
	}
	return v
}

// exeInfo is the file a process runs.
type exeInfo struct {
	path string
	gone bool // removed, or another file is at path now
}

// runningExes maps each pid to the file it runs: /proc on Linux, one lsof
// call on macOS (the first txt entry is the executable).
func runningExes(ctx context.Context, goos string, procs []runningProc) map[int]exeInfo {
	out := map[int]exeInfo{}
	if len(procs) == 0 {
		return out
	}
	if goos == "linux" {
		for _, p := range procs {
			link := fmt.Sprintf("/proc/%d/exe", p.PID)
			path, err := os.Readlink(link)
			if err != nil {
				continue
			}
			gone := strings.HasSuffix(path, " (deleted)")
			path = strings.TrimSuffix(path, " (deleted)")
			if !gone {
				a, e1 := os.Stat(link)
				b, e2 := os.Stat(path)
				gone = e1 != nil || e2 != nil || !os.SameFile(a, b)
			}
			out[p.PID] = exeInfo{path, gone}
		}
		return out
	}
	pids := make([]string, len(procs))
	for i, p := range procs {
		pids[i] = strconv.Itoa(p.PID)
	}
	b, _ := exec.CommandContext(ctx, "lsof", "-a", "-p", strings.Join(pids, ","), "-d", "txt", "-Fpin").Output()
	for pid, t := range parseLsofExe(b) {
		gone := !filepath.IsAbs(t.name) // macOS names a removed file by its base name
		if !gone {
			gone = fileInode(t.name) != t.inode
		}
		out[pid] = exeInfo{t.name, gone}
	}
	return out
}

type lsofTxt struct {
	name  string
	inode uint64
}

// parseLsofExe reads lsof -d txt -Fpin: per p line, the first txt file
// (its i and n lines) is the executable.
func parseLsofExe(b []byte) map[int]lsofTxt {
	out := map[int]lsofTxt{}
	pid := 0
	var ino uint64
	for _, line := range strings.Split(string(b), "\n") {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
			ino = 0
		case 'i':
			ino, _ = strconv.ParseUint(line[1:], 10, 64)
		case 'n':
			if _, ok := out[pid]; !ok && pid != 0 {
				out[pid] = lsofTxt{line[1:], ino}
			}
		}
	}
	return out
}

// runningGOOS is the system versionRunning lists for; tests set another.
var runningGOOS = runtime.GOOS

// runVersion is hug version [--running].
func runVersion(args []string) int {
	fmt.Printf("hug %s\n", Version)
	switch strings.Join(args, " ") {
	case "":
		return 0
	case "--running":
		return versionRunning(os.Stdout, runningGOOS)
	}
	fmt.Fprintf(os.Stderr, "hug version: unknown argument %q\nfix: hug version --running\n", strings.Join(args, " "))
	return 2
}

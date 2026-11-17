package task

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/veridit/hug/internal/cond"
)

// Wait is a condition a task waits on (16dxsr): a Depends on line
// "- when PREDICATE", or "- group:NAME" for "when group NAME closed". The
// predicate is any cond.Parse reads: a date, the load below N, a branch
// merged, a tag or release present, a task or group closed. Wakes names who
// is told when it holds; empty means whoever has the task taken.
type Wait struct {
	// Text is the predicate as written, without "when" (canonical when it
	// parses: "load below 120", "branch feat/x merged at 1a2b3c").
	Text  string `json:"text"`
	Line  int    `json:"line"`
	Wakes string `json:"wakes,omitempty"`
}

// WaitDetail is a wait as show, MCP and the TUI report it: met or not, and
// what hug saw when it checked.
type WaitDetail struct {
	Text  string `json:"text"`
	Line  int    `json:"line"`
	Met   bool   `json:"met"`
	Seen  string `json:"seen"`
	Wakes string `json:"wakes,omitempty"`
	// Board: the condition is about the board (a task or group closed), so
	// it holds back done as a dependency does; the others (date, load,
	// branch, tag) only hold back starting: next and take.
	Board bool `json:"board,omitempty"`
	// Unknown: hug cannot check it here (unparseable, no git, no load
	// average): never met, and lint names it.
	Unknown bool `json:"unknown,omitempty"`
}

var (
	waitLine  = regexp.MustCompile(`(?i)^when\s+(.+?)(?:,?\s+wakes\s+([A-Za-z0-9][A-Za-z0-9._@\-]*))?$`)
	groupLine = regexp.MustCompile(`(?i)^group:([A-Za-z0-9][A-Za-z0-9._/\-]*)(?:,?\s+wakes\s+([A-Za-z0-9][A-Za-z0-9._@\-]*))?$`)
)

// parseWaitLine reads a Depends on line that states a condition. ok is
// false for anything else (a dependency or prose). A "when" line whose
// predicate hug cannot read is still a wait: it never holds, and lint
// reports it (the principle: an unknown condition counts as unmet).
func parseWaitLine(line string) (w Wait, err error, ok bool) {
	t := strings.TrimSpace(line)
	for _, bullet := range []string{"- ", "* "} {
		if strings.HasPrefix(t, bullet) {
			t = strings.TrimSpace(t[len(bullet):])
			break
		}
	}
	t = strings.Join(strings.Fields(t), " ")
	if m := groupLine.FindStringSubmatch(t); m != nil {
		return Wait{Text: "group " + m[1] + " closed", Wakes: m[2]}, nil, true
	}
	m := waitLine.FindStringSubmatch(t)
	if m == nil {
		return Wait{}, nil, false
	}
	w = Wait{Text: m[1], Wakes: m[2]}
	c, err := cond.Parse(m[1])
	if err == nil {
		w.Text = c.Written()
	}
	return w, err, true
}

// The world is what hug knows outside the board when it checks a wait: the
// machine's load and the git checkout's branches and tags. The command
// sets it for its checkout (SetWorld, cli: cond.Local); the default knows the date
// and the load only, so branch and tag waits are unknown.
var world struct {
	sync.RWMutex
	f func() cond.State
}

// SetWorld sets what waits are checked against; nil restores the default.
func SetWorld(f func() cond.State) {
	world.Lock()
	world.f = f
	world.Unlock()
}

func worldState() cond.State {
	world.RLock()
	f := world.f
	world.RUnlock()
	if f == nil {
		return cond.State{Load: func() (float64, bool) { return cond.LoadAverage() }}
	}
	return f()
}

// boardState answers task and group conditions from the board itself.
func boardState(st cond.State, tasks []Task) cond.State {
	var once sync.Once
	var groups map[string][2]int // group -> total, open
	st.TaskOpen = func(id string) (bool, bool) {
		x, e := ResolveDependency(tasks, id)
		if e != nil {
			return false, false
		}
		return !Closed(x, tasks), true
	}
	st.GroupOpen = func(name string) (int, bool) {
		once.Do(func() {
			groups = map[string][2]int{}
			for _, t := range tasks {
				for g := t.Group; g != ""; {
					c := groups[strings.ToLower(g)]
					c[0]++
					if t.Status != "done" && t.Status != "dropped" {
						c[1]++
					}
					groups[strings.ToLower(g)] = c
					i := strings.LastIndex(g, "/")
					if i < 0 {
						break
					}
					g = g[:i]
				}
			}
		})
		c := groups[strings.ToLower(strings.Trim(name, "/"))]
		return c[1], c[0] > 0
	}
	return st
}

// Closed reports whether a dependency on t is settled: t is done or
// dropped, or t is an epic whose children are all done or dropped (16dxsr:
// depending on an epic waits for its whole content, not its own status).
func Closed(t Task, tasks []Task) bool {
	if t.Status == "done" || t.Status == "dropped" {
		return true
	}
	kids := childPositions(tasks, t)
	if len(kids) == 0 {
		return false
	}
	for _, i := range kids {
		if s := tasks[i].Status; s != "done" && s != "dropped" {
			return false
		}
	}
	return true
}

// The last board's epic -> child positions, keyed like refIndexFor: Blocked
// runs for every task on a 10,000-task board (TUI, board), so the children
// are found once per board, not once per dependency. Statuses are read
// from the board itself, so only a parent change (a write, which reloads
// the board) could make it stale; positions are checked on use.
var childMemo struct {
	sync.Mutex
	first *Task
	n     int
	idx   map[string][]int
}

func childPositions(tasks []Task, t Task) []int {
	scan := func() []int {
		var out []int
		for i, c := range tasks {
			if c.Parent != "" && (strings.EqualFold(c.Parent, t.ID) || containsFold(t.Aliases, c.Parent)) {
				out = append(out, i)
			}
		}
		return out
	}
	if len(tasks) < 64 {
		return scan()
	}
	childMemo.Lock()
	defer childMemo.Unlock()
	build := func() {
		idx := map[string][]int{}
		byRef := map[string]string{}
		for _, x := range tasks {
			byRef[strings.ToLower(x.ID)] = strings.ToLower(x.ID)
		}
		for _, x := range tasks {
			for _, a := range x.Aliases {
				if _, taken := byRef[strings.ToLower(a)]; !taken {
					byRef[strings.ToLower(a)] = strings.ToLower(x.ID)
				}
			}
		}
		for i, c := range tasks {
			if c.Parent == "" {
				continue
			}
			if p, ok := byRef[strings.ToLower(c.Parent)]; ok {
				idx[p] = append(idx[p], i)
			}
		}
		childMemo.first, childMemo.n, childMemo.idx = &tasks[0], len(tasks), idx
	}
	if childMemo.first != &tasks[0] || childMemo.n != len(tasks) {
		build()
	}
	out := childMemo.idx[strings.ToLower(t.ID)]
	for _, i := range out {
		if i >= len(tasks) || !(strings.EqualFold(tasks[i].Parent, t.ID) || containsFold(t.Aliases, tasks[i].Parent)) {
			build()
			return childMemo.idx[strings.ToLower(t.ID)]
		}
	}
	return out
}

// Waits checks t's waits now, in file order.
func Waits(t Task, tasks []Task) []WaitDetail {
	if len(t.Waits) == 0 {
		return nil
	}
	st := boardState(worldState(), tasks)
	// A task in the group it waits on does not wait on itself.
	if inner := st.GroupOpen; inner != nil {
		st.GroupOpen = func(name string) (int, bool) {
			n, known := inner(name)
			g := strings.ToLower(strings.Trim(name, "/"))
			mine := strings.ToLower(t.Group)
			if known && n > 0 && t.Status != "done" && t.Status != "dropped" && (mine == g || strings.HasPrefix(mine, g+"/")) {
				n--
			}
			return n, known
		}
	}
	out := make([]WaitDetail, 0, len(t.Waits))
	for _, w := range t.Waits {
		d := WaitDetail{Text: w.Text, Line: w.Line, Wakes: w.Wakes}
		if d.Wakes == "" {
			d.Wakes = strings.Join(t.Assignee, ", ")
		}
		c, err := cond.Parse(w.Text)
		if err != nil {
			d.Seen, d.Unknown = "unknown: hug cannot check this condition", true
		} else {
			d.Met, d.Seen = c.Met(st)
			d.Unknown = !d.Met && strings.HasPrefix(d.Seen, "unknown:")
			d.Board = c.Kind == cond.Task || c.Kind == cond.Group
		}
		out = append(out, d)
	}
	return out
}

// Unmet is the waits that do not hold now; with board only, those about the
// board (task, group), which hold back done.
func Unmet(t Task, tasks []Task, board bool) []WaitDetail {
	var out []WaitDetail
	for _, w := range Waits(t, tasks) {
		if !w.Met && (!board || w.Board) {
			out = append(out, w)
		}
	}
	return out
}

// Waiting reports whether t cannot start now: an open dependency or a wait
// that does not hold. Next skips such a task; nobody polls.
func Waiting(t Task, tasks []Task) bool {
	return len(Blocked(t, tasks)) > 0 || len(Unmet(t, tasks, false)) > 0
}

// WaitFindings names, for open tasks, each wait hug cannot check here: a
// branch this clone does not have, no load average on this system. It
// reads git and the machine, so like BranchFindings it is never cached.
// Unparseable waits are reported by the parse (unknown-condition).
func WaitFindings(ts []Task) []Finding {
	var out []Finding
	for _, t := range ts {
		if len(t.Waits) == 0 || t.Status == "done" || t.Status == "dropped" {
			continue
		}
		for _, w := range Waits(t, ts) {
			if !w.Unknown {
				continue
			}
			if _, err := cond.Parse(w.Text); err != nil {
				continue // the parse finding says so
			}
			out = append(out, Finding{Path: t.Path, Line: w.Line, Code: "unknown-condition",
				Problem: fmt.Sprintf("unknown condition: %q cannot be checked here (%s), so %s waits", "when "+w.Text, strings.TrimPrefix(w.Seen, "unknown: "), t.ID),
				Fix:     "hug task dep " + t.ID + " --remove " + shellQuote("when "+w.Text)})
		}
	}
	return out
}

// unknownCondition is the parse finding for a "when" line hug cannot read.
func unknownCondition(id, text string, line int, path string, err error) Finding {
	who := id
	if who == "" {
		who = "ID"
	}
	return Finding{Path: path, Line: line, Code: "unknown-condition",
		Problem: "unknown condition: " + quoteLine("when "+text, 60) + ": " + err.Error() + "; until it is fixed the task waits",
		Fix:     "hug task dep " + who + " --remove " + shellQuote("when "+text)}
}

func shellQuote(s string) string {
	if !strings.ContainsAny(s, " \t'\"\\$`!*?;&|<>()[]{}#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// WaitSummary is the unmet waits in one line: "when load below 120 (load is
// 140.2); when group auth closed (3 open in group auth)".
func WaitSummary(ws []WaitDetail) string {
	parts := make([]string, 0, len(ws))
	for _, w := range ws {
		parts = append(parts, "when "+w.Text+" ("+w.Seen+")")
	}
	return strings.Join(parts, "; ")
}

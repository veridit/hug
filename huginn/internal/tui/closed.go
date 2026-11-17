package tui

import (
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// closeLog knows when tasks were closed, for RECENTLY DONE (zaec08). The
// updated date is set only when a task is created, so the close time comes
// from git: the commit time of "hug task done|drop ID", or of a commit whose
// diff turns a task's status from open to done or dropped (set, edit). A
// commit that only touches a task already closed does not count. The log is
// read once per branch head, then only the commits since the last head.
type closeLog struct {
	mu   sync.Mutex
	head string
	at   map[string]int64 // lowercase id -> unix close time; never mutated once published
}

const (
	closeScanFirst = 200 // diffs read on the first load; subjects are read for all history
	closeScanMore  = 500 // diffs read for the commits since the last head
)

var closeSubject = regexp.MustCompile(`^(?:hug task|huginn) (?:done|drop) (\S+)$`)

func git(root string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", root}, args...)...).Output()
	return string(out), err
}

// times returns the close times known at the current head; nil without git.
func (c *closeLog) times(root string) map[string]int64 {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out, err := git(root, "rev-parse", "-q", "--verify", "HEAD")
	head := strings.TrimSpace(out)
	if err != nil || head == "" {
		return c.at
	}
	if head == c.head {
		return c.at
	}
	format := "--format=\x01%ct %s"
	next := map[string]int64{}
	for id, at := range c.at {
		next[id] = at
	}
	found := false
	if c.head != "" {
		if log, err := git(root, "log", "--no-color", "--no-renames", "-p", "-U0", "-n", strconv.Itoa(closeScanMore), format, c.head+".."+head); err == nil {
			mergeCloses(next, log)
			found = true
		}
	}
	if !found {
		// First load, or the old head is gone (history rewritten).
		next = map[string]int64{}
		if log, err := git(root, "log", "--no-color", "-E", "--grep=^(hug task|huginn) (done|drop) ", format, head); err == nil {
			mergeCloses(next, log)
		}
		if log, err := git(root, "log", "--no-color", "--no-renames", "-p", "-U0", "-n", strconv.Itoa(closeScanFirst), format, head); err == nil {
			mergeCloses(next, log)
		}
	}
	c.head, c.at = head, next
	return next
}

// stateValue is the value of a "state: x" or "status: x" line.
func stateValue(line string) string {
	_, v, _ := strings.Cut(line, ":")
	return v
}

func closedValue(v string) bool {
	v = strings.Trim(strings.TrimSpace(v), `"'`)
	return v == "done" || v == "dropped"
}

// mergeCloses reads git log output (format "\x01<unix> <subject>", with or
// without -p -U0 diffs) and keeps the newest close time per task.
func mergeCloses(into map[string]int64, log string) {
	var at int64
	file, opened := "", false
	record := func(id string) {
		id = strings.ToLower(id)
		if at > into[id] {
			into[id] = at
		}
	}
	for _, line := range strings.Split(log, "\n") {
		switch {
		case strings.HasPrefix(line, "\x01"):
			stamp, subject, _ := strings.Cut(line[1:], " ")
			at, _ = strconv.ParseInt(stamp, 10, 64)
			if m := closeSubject.FindStringSubmatch(subject); m != nil {
				record(m[1])
			}
			file, opened = "", false
		case strings.HasPrefix(line, "diff --git "):
			file, opened = "", false
		case strings.HasPrefix(line, "+++ "):
			file = strings.TrimPrefix(strings.TrimPrefix(line, "+++ "), "b/")
		// History holds both keys: status: before hug upgrade, state: after
		// (6ephqk step 4). The upgrade's own -status:/+state: pair keeps
		// the value, so it closes nothing.
		case strings.HasPrefix(line, "-status:"), strings.HasPrefix(line, "-state:"):
			opened = !closedValue(stateValue(line[1:]))
		case strings.HasPrefix(line, "+status:"), strings.HasPrefix(line, "+state:"):
			if opened && file != "" && closedValue(stateValue(line[1:])) {
				if _, id, _, err := task.ParseName(file); err == nil {
					record(id)
				}
			}
		}
	}
}

// recentIndices are the visible done and dropped tasks, closed most
// recently first (zaec08); all of them, the focused section scrolls
// (h9zghj). Ties keep rank order.
func (m model) recentIndices() []int {
	var out []int
	taskScans.Add(int64(len(m.visible)))
	for i, t := range m.visible {
		if t.State == "done" || t.State == "dropped" {
			out = append(out, i)
		}
	}
	keys := make(map[int]int64, len(out))
	for _, i := range out {
		keys[i] = m.closedKey(m.visible[i])
	}
	sort.SliceStable(out, func(a, b int) bool { return keys[out[a]] > keys[out[b]] })
	return out
}

// closedKey orders closed tasks for RECENTLY DONE: the close time from git,
// else the updated date (imported boards carry it), else nothing.
func (m model) closedKey(t task.Task) int64 {
	if at := m.closedAt[strings.ToLower(t.ID)]; at > 0 {
		return at
	}
	v := strings.Trim(strings.TrimSpace(t.Updated), `"'`)
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04", "2006-01-02T15:04", "2006-01-02"} {
		if at, err := time.ParseInLocation(layout, v, time.Local); err == nil {
			return at.Unix()
		}
	}
	return 0
}

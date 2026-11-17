// Package cond is the one vocabulary of conditions hug can check itself,
// without a model call or the network (16dxsr, shared with ez50rb's memory
// conditions): a date reached, the load below N, a branch merged, a tag or
// release present, a task closed, every task in a group closed. A task
// waits on one in its Depends on section ("- when load below 120"); a
// memory holds while one has not yet happened ("while branch X unmerged").
//
// A condition is met, not met, or unknown. Unknown (no git here, a branch
// this clone has never seen, no load average on this system) is never
// met: a wait on it keeps waiting, and the reading says why.
package cond

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Kinds of condition.
const (
	Date   = "date"   // the day has come: Date
	Load   = "load"   // the 5-minute load average is below N
	Branch = "branch" // branch Ref is merged into the main branch (At: its tip when the wait was written)
	Tag    = "tag"    // tag Ref exists (a release is a tag)
	Task   = "task"   // task Ref is done or dropped
	Group  = "group"  // every task in group Ref (and its subgroups) is done or dropped
)

// Kinds lists the predicates Parse reads, as they are written.
const Kinds = "date YYYY-MM-DD, load below N, branch NAME merged, release NAME exists, tag NAME exists, task:ID closed, group NAME closed"

// Cond is one parsed condition. Text is its canonical form, as stored.
type Cond struct {
	Kind string
	Ref  string    // Branch, Tag, Task, Group: the name or id
	N    float64   // Load: the threshold
	Date time.Time // Date: the first day it is met
	// At is the branch tip recorded when the wait was written, so a branch
	// merged and then deleted still reads as merged.
	At   string
	Text string
}

var (
	dateRe   = regexp.MustCompile(`^(?:date )?(\d{4}-\d{2}-\d{2})$`)
	loadRe   = regexp.MustCompile(`^load (?:below|under|<) ([0-9]+(?:\.[0-9]+)?)$`)
	branchRe = regexp.MustCompile(`^(?i:branch) ([A-Za-z0-9][A-Za-z0-9._/\-]*) (?i:(?:is )?merged(?: at ([0-9a-f]{7,40}))?)$`)
	tagRe    = regexp.MustCompile(`^(?i:(tag|release)) ([A-Za-z0-9][A-Za-z0-9._/\-+]*) (?i:exists|is published|published)$`)
	taskRe   = regexp.MustCompile(`^task[: ]([A-Za-z0-9][A-Za-z0-9._\-]*) (?:is )?(?:closed|done)$`)
	groupRe  = regexp.MustCompile(`^(?i:group)[: ]([A-Za-z0-9][A-Za-z0-9._/\-]*)(?i:(?: (?:is )?(?:closed|done))?)$`)
)

// Parse reads one predicate, without the leading "when".
func Parse(s string) (Cond, error) {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	low := strings.ToLower(s)
	if m := dateRe.FindStringSubmatch(low); m != nil {
		d, err := time.ParseInLocation("2006-01-02", m[1], time.Local)
		if err != nil {
			return Cond{}, fmt.Errorf("%s is not a date", m[1])
		}
		return Cond{Kind: Date, Date: d, Text: "date " + m[1]}, nil
	}
	if m := loadRe.FindStringSubmatch(low); m != nil {
		n, _ := strconv.ParseFloat(m[1], 64)
		return Cond{Kind: Load, N: n, Text: "load below " + m[1]}, nil
	}
	if m := branchRe.FindStringSubmatch(s); m != nil {
		return Cond{Kind: Branch, Ref: m[1], At: strings.ToLower(m[2]), Text: "branch " + m[1] + " merged"}, nil
	}
	if m := tagRe.FindStringSubmatch(s); m != nil {
		return Cond{Kind: Tag, Ref: m[2], Text: strings.ToLower(m[1]) + " " + m[2] + " exists"}, nil
	}
	if m := taskRe.FindStringSubmatch(low); m != nil {
		return Cond{Kind: Task, Ref: m[1], Text: "task:" + m[1] + " closed"}, nil
	}
	if m := groupRe.FindStringSubmatch(s); m != nil {
		return Cond{Kind: Group, Ref: m[1], Text: "group " + m[1] + " closed"}, nil
	}
	return Cond{}, fmt.Errorf("%q is not a condition hug can check (it can check: %s)", s, Kinds)
}

// Written is the condition as a Depends on line writes it: Text, plus the
// recorded branch tip.
func (c Cond) Written() string {
	if c.Kind == Branch && c.At != "" {
		return c.Text + " at " + c.At
	}
	return c.Text
}

// State is what a checker knows. A nil function is unknown.
type State struct {
	Now time.Time // zero: time.Now()
	// Load is the 5-minute load average.
	Load func() (avg float64, known bool)
	// Branch reports whether a branch is merged into the main branch; at is
	// the tip recorded with the condition, if any. why says what was seen.
	Branch func(name, at string) (merged bool, why string, known bool)
	// Tag reports whether a tag exists.
	Tag func(name string) (exists, known bool)
	// TaskOpen reports a task: known false when it is done or dropped.
	TaskOpen func(id string) (open, known bool)
	// GroupOpen counts the open tasks in a group; known false when the
	// group has no task at all.
	GroupOpen func(name string) (open int, known bool)
}

// Met reports whether c holds now, and what was seen ("load is 23.5",
// "branch feat/x is not merged into main", "unknown: no load average").
func (c Cond) Met(st State) (met bool, seen string) {
	now := st.Now
	if now.IsZero() {
		now = time.Now()
	}
	unknown := func(what string) (bool, string) { return false, "unknown: " + what }
	switch c.Kind {
	case Date:
		if !now.Before(c.Date) {
			return true, "the date " + c.Date.Format("2006-01-02") + " has come"
		}
		return false, "today is " + now.Format("2006-01-02")
	case Load:
		if st.Load == nil {
			return unknown("no load average here")
		}
		avg, ok := st.Load()
		if !ok {
			return unknown("no load average on this system")
		}
		return avg < c.N, "load is " + strconv.FormatFloat(avg, 'f', 1, 64)
	case Branch:
		if st.Branch == nil {
			return unknown("not in a git checkout")
		}
		merged, why, ok := st.Branch(c.Ref, c.At)
		if !ok {
			return unknown(why)
		}
		return merged, why
	case Tag:
		if st.Tag == nil {
			return unknown("not in a git checkout")
		}
		ok, known := st.Tag(c.Ref)
		switch {
		case !known:
			return unknown("not in a git checkout")
		case ok:
			return true, "tag " + c.Ref + " exists"
		}
		return false, "no tag " + c.Ref + " here yet (git fetch --tags fetches new ones)"
	case Task:
		if st.TaskOpen == nil {
			return unknown("no board")
		}
		open, known := st.TaskOpen(c.Ref)
		switch {
		case !known:
			return unknown("no task " + c.Ref)
		case open:
			return false, "task " + c.Ref + " is open"
		}
		return true, "task " + c.Ref + " is closed"
	case Group:
		if st.GroupOpen == nil {
			return unknown("no board")
		}
		n, known := st.GroupOpen(c.Ref)
		switch {
		case !known:
			return unknown("no task is in group " + c.Ref)
		case n > 0:
			return false, fmt.Sprintf("%d open in group %s", n, c.Ref)
		}
		return true, "every task in group " + c.Ref + " is closed"
	}
	return unknown("not a condition hug can check")
}

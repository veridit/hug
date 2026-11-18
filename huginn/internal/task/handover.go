package task

import (
	"strings"
	"time"
)

// Handing a gone worker's tasks back (yf39ya check 4, Jørgen 2026-11-18:
// 17 stale-owner findings after workers' sessions ended): a doing task
// whose every assignee is a session this machine knows and that has gone
// by the stale-owner rule (closed over GoneOwnerAfter ago, or silent over
// StaleOwnerAfter) is given to the board's coordinator, its stage kept, so
// someone who is here owns it. Only the background sync pass and hug task
// sync do it (crab 20:09: lint never writes). A person, a model name and a
// name no local session knows (it may be live on another machine) are
// never handed over: stale-owner keeps reporting those, as it does when
// there is no coordinator or the coordinator has gone too.

// Handover is one task to hand to the coordinator.
type Handover struct {
	ID, From, To string
	// Seen is when the last of From's sessions was seen; Closed whether it
	// ended (rather than went silent).
	Seen   time.Time
	Closed bool
}

// Line is the History reason: "handed to crab: bonehound's session ended
// (closed 2026-11-18 21:03)".
func (h Handover) Line() string {
	how := "went silent (last seen "
	if h.Closed {
		how = "ended (closed "
	}
	return "handed to " + h.To + ": " + h.From + "'s session " + how + h.Seen.Local().Format("2006-01-02 15:04") + ")"
}

// coordinatorHere reports whether the coordinator can take tasks: a person
// on the board, or a session this machine knows that has not gone.
func coordinatorHere(live *Liveness, name string, now time.Time, people []Person) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	if _, ok := PersonByName(people, name); ok {
		return true
	}
	gone, known := live.Gone(name, now)
	return known && !gone
}

// PlanHandover lists the doing tasks to hand to coordinator (see above).
func PlanHandover(live *Liveness, ts []Task, coordinator string, now time.Time, people ...Person) []Handover {
	if !live.Known() || !coordinatorHere(live, coordinator, now, people) {
		return nil
	}
	var out []Handover
	for _, t := range ts {
		if t.State != "doing" || len(t.Assignee) == 0 || t.Model == "human" {
			continue
		}
		h := Handover{ID: t.ID, To: coordinator}
		all := true
		for _, a := range t.Assignee {
			if strings.EqualFold(strings.TrimSpace(a), coordinator) {
				all = false
				break
			}
			if _, isPerson := PersonByName(people, a); isPerson || isModelName(a, t) {
				all = false
				break
			}
			s, _, ok := live.ownerSeen(a)
			if !ok || !sessionGone(s, now) {
				all = false
				break
			}
			if s.Seen.After(h.Seen) {
				h.Seen, h.Closed = s.Seen, s.Closed
			}
		}
		if !all {
			continue
		}
		h.From = strings.Join(t.Assignee, ", ")
		out = append(out, h)
	}
	return out
}

// HandOver gives h.ID to h.To, keeping its stage, with one History line.
// A task that changed meanwhile (not doing, or its assignee not h.From) is
// left alone.
func (s *Store) HandOver(h Handover) (Result, error) {
	d, _, e := s.load(h.ID)
	if e != nil {
		return Result{}, e
	}
	if d.Task.State != "doing" || strings.Join(d.Task.Assignee, ", ") != h.From {
		return s.finish(d, false)
	}
	if e := d.SetField("assignee", []string{h.To}); e != nil {
		return Result{}, e
	}
	d.History(stamp() + " " + h.Line())
	return s.finish(d, true)
}

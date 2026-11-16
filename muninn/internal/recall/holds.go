package recall

import (
	"time"

	"github.com/veridit/hug/muninn/internal/holds"
	"github.com/veridit/hug/muninn/internal/store"
)

// holdsEnded reports whether u carries a stated condition (ez50rb) that no
// longer holds: a date that has passed (checked here, no I/O), or a task or
// branch condition the dream pass found ended (holds_ended). A unit with
// no condition is never filtered, so recall is unchanged until a memory
// carries one.
func (e *Engine) holdsEnded(u store.Unit, now time.Time) bool {
	if u.Holds == "" {
		return false
	}
	c, err := holds.Parse(u.Holds)
	if err != nil || c.Kind == "" {
		return false // a condition hug cannot read is not an end
	}
	if ended, _ := c.Ended(holds.State{Now: now}); ended {
		return true
	}
	if e.DB == nil || u.SID == "" {
		return false
	}
	var one int
	return e.DB.QueryRow(`SELECT 1 FROM holds_ended WHERE sid=? AND holds=?`, u.SID, c.Text).Scan(&one) == nil
}

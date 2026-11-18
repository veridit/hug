package cli

import (
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// wakeDue ends every timed wait whose date has come (z2vmte): any hug read
// does it, so no agent has to remember. One commit for all of them; a
// command that is no write takes the board turn for it, and only when a
// date has come (a scan of the board in memory otherwise). It reports
// nothing on failure: the next read tries again, and the wait shows its
// date meanwhile.
func (s *session) wakeDue(ts []task.Task, holdsTurn bool) {
	due := task.Due(ts)
	if len(due) == 0 {
		return
	}
	if !holdsTurn {
		defer s.turn()()
		s.s.Invalidate()
	}
	var woke []string
	for _, id := range due {
		if r, e := s.s.Wake(id); e == nil && r.Changed {
			woke = append(woke, id)
		}
	}
	if len(woke) > 0 {
		_ = s.commit("hug task wake: " + strings.Join(woke, ", ") + " (the date came)")
	}
}

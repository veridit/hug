package task

import (
	"testing"
	"time"
)

// w40cmk (hatchling's ask): a task-branch commit that changes C/ID.md marks
// the ticket, as one that changed rank.id.slug.md did; a group's old path
// and a file in the task's folder do too, and a non-task file does not.
func TestTouchedSinceLayout3(t *testing.T) {
	got := TouchedSince([]string{
		"1700000000\tedit\t3/37bryg.md",
		"1700000100\tedit\tlater/00i.6ephqk.task-state-stage-hard.md",
		"1700000200\tedit\thug.toml\tx/notanid.md\t3/abcdef.md",
	})
	for id, sec := range map[string]int64{"37bryg": 1700000000, "6ephqk": 1700000100} {
		if !got[id].Equal(time.Unix(sec, 0)) {
			t.Errorf("%s touched %v, want %v (all: %v)", id, got[id], time.Unix(sec, 0), got)
		}
	}
	if _, ok := got["abcdef"]; ok {
		t.Errorf("3/abcdef.md is not C/ID.md for abcdef (shard 3 is not a), but it was marked: %v", got)
	}
	if len(got) != 2 {
		t.Errorf("marked %d tickets, want 2: %v", len(got), got)
	}
}

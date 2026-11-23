package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// atex3d (z2vmte check 5, Jørgen 2026-11-20 13:05): show's stage line for
// a wait with a date leads with the date: "waiting until 2026-11-24 · on
// WHY · since ...", "waiting until 2026-11-24 · then ask NAME · since
// ...", with no second "until" and no "on the date". The stored fields
// stay as v0.7.55 wrote them, so an old task reads the new way too. Not
// parallel: it sets the clock.
func TestShowLeadsADatedWaitWithTheDate(t *testing.T) {
	old := task.Now
	t.Cleanup(func() { task.Now = old })
	task.Now = func() time.Time { return time.Date(2026, 11, 18, 5, 8, 0, 0, time.Local) }
	a := app(t)
	execute(t, a, "init")
	newTask := func(title string) string {
		return execute(t, a, "new", title, "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a")["task"].(Reply)["id"].(string)
	}
	w, q := newTask("Look again"), newTask("Ask later")
	execute(t, a, "take", w, "--as", "kestrel", "--model", "m")
	execute(t, a, "wait", w, "--until", "2026-11-24", "a week of use", "--as", "kestrel")
	execute(t, a, "take", q, "--as", "crab", "--model", "m")
	execute(t, a, "ask", q, "--to", "jhf", "--after", "2026-11-24", "--check", "1", "has he had to remind an agent?", "--as", "crab")
	if f := taskFile(t, a, q); !strings.Contains(f, `waiting_on: "the date 2026-11-24, then ask jhf"`) || !strings.Contains(f, `waiting_until: "2026-11-24"`) {
		t.Fatalf("stored fields changed:\n%s", f)
	}
	for id, want := range map[string]string{
		w: "stage: waiting until 2026-11-24 · on a week of use · since 2026-11-18 05:08",
		q: "stage: waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08",
	} {
		_, out := runText(a, "show", id)
		if got := strings.Split(out, "\n")[1]; got != want {
			t.Errorf("show %s line 2:\n got %q\nwant %q", id, got, want)
		}
	}
}

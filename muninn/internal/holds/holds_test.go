package holds

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	for in, want := range map[string]string{
		"":                             "",
		"until 2026-12-01":             "until 2026-12-01",
		"  Until   2026-12-01 ":        "until 2026-12-01",
		"while task:ez50rb open":       "while task:ez50rb open",
		"while task:TASK-1960 is open": "while task:task-1960 open",
		"while branch feat/x unmerged": "while branch feat/x unmerged",
		"while feat/Y is unmerged":     "while branch feat/Y unmerged",
	} {
		c, err := Parse(in)
		if err != nil || c.Text != want {
			t.Errorf("Parse(%q) = %q, %v; want %q", in, c.Text, err, want)
		}
	}
	for _, in := range []string{"until tomorrow", "until 2026-13-40", "while dragon is running", "forever"} {
		if _, err := Parse(in); err == nil || !strings.Contains(err.Error(), "\nfix: ") {
			t.Errorf("Parse(%q): want error ending in a fix line, got %v", in, err)
		}
	}
}

func TestEnded(t *testing.T) {
	day := func(s string) time.Time { d, _ := time.ParseInLocation("2006-01-02", s, time.Local); return d }
	st := State{
		Now:        day("2026-12-01").Add(time.Hour),
		TaskOpen:   func(id string) (bool, bool) { return id == "open01", id != "nobody" },
		BranchLive: func(n string) (bool, bool) { return n == "feat/live", n != "unknown" },
	}
	for in, want := range map[string]bool{
		"until 2026-12-01":                true,
		"until 2026-12-02":                false,
		"while task:open01 open":          false,
		"while task:done01 open":          true,
		"while task:nobody open":          false, // unknown is never an end
		"while branch feat/live unmerged": false,
		"while branch feat/gone unmerged": true,
		"while branch unknown unmerged":   false,
		"":                                false,
	} {
		c, err := Parse(in)
		if err != nil {
			t.Fatal(err)
		}
		if got, why := c.Ended(st); got != want || (got && why == "") {
			t.Errorf("%q: ended=%v (%s), want %v", in, got, why, want)
		}
	}
}

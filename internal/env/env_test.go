package env

import (
	"strings"
	"testing"
)

// m1c28r check 3: an old name set is named with its HUG_ name and a fix:
// line. The old names are spelt in parts: the code holds none.
func TestStaleNamesWarning(t *testing.T) {
	home, user := "MUN"+"INN_HOME", "HUG"+"INN_USER"
	old := Stale([]string{"PATH=/bin", user + "=poodle", home + "=/tmp/m", "HUG_MODEL=x", "MUN" + "INN_=odd"})
	if len(old) != 2 || old[0] != (Old{user, User}) || old[1] != (Old{home, Home}) {
		t.Fatalf("Stale = %v", old)
	}
	problem, fix := Warning(old)
	for _, want := range []string{user + " is now HUG_USER", home + " is now HUG_HOME", "the old name is ignored"} {
		if !strings.Contains(problem, want) {
			t.Errorf("problem %q lacks %q", problem, want)
		}
	}
	if want := `export HUG_USER="$` + user + `" HUG_HOME="$` + home + `"; unset ` + user + " " + home; !strings.HasPrefix(fix, want) {
		t.Errorf("fix %q, want prefix %q", fix, want)
	}
	if p, f := Warning(Stale([]string{"HUG_HOME=/x"})); p != "" || f != "" {
		t.Errorf("no old names, still %q %q", p, f)
	}
}

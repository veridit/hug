package cli

import (
	"regexp"
	"strings"
	"testing"
)

// d8yx7y check 4: hug task ls shows ticked/total acceptance checks per task,
// after the type, so the id stays the first field.
func TestLsShowsCheckProgress(t *testing.T) {
	t.Parallel()
	a := app(t)
	two := execute(t, a, "new", "Two checks", "--goal", "g", "--purpose", "P", "--principle", "Q", "--accept", "first", "--accept", "second")["task"].(Reply)["id"].(string)
	execute(t, a, "check", two, "1")
	one := execute(t, a, "new", "One check", "--goal", "g", "--purpose", "P", "--principle", "Q", "--accept", "only")["task"].(Reply)["id"].(string)
	code, out := runText(a, "ls")
	if code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	for id, want := range map[string]string{two: "1/2", one: "0/1"} {
		row := regexp.MustCompile(`(?m)^` + id + `\b.*$`).FindString(out)
		if f := strings.Fields(row); len(f) < 4 || f[0] != id || f[3] != want {
			t.Fatalf("%s: want %q as field 4 in %q:\n%s", id, want, row, out)
		}
	}
}

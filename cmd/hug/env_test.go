package main

import (
	"strings"
	"testing"
)

// m1c28r check 3: with an old name set, hug says which HUG_ name to use,
// with a fix: line, on stderr; nothing without one. Old names are spelt in
// parts: the code holds none.
func TestWarnStale(t *testing.T) {
	old := "MUN" + "INN_HOME"
	var b strings.Builder
	warnStale(&b, []string{old + "=/tmp/m", "PATH=/bin"})
	got := b.String()
	if !strings.Contains(got, "hug: hug reads only HUG_* variables: "+old+" is now HUG_HOME") || !strings.Contains(got, "\nfix: export HUG_HOME=\"$"+old+"\"; unset "+old) {
		t.Fatalf("warning:\n%s", got)
	}
	b.Reset()
	warnStale(&b, []string{"HUG_HOME=/tmp/m"})
	if b.Len() != 0 {
		t.Fatalf("warned without old names: %q", b.String())
	}
}

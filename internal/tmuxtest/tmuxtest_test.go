package tmuxtest

import (
	"os"
	"reflect"
	"runtime"
	"testing"
)

// Reap picks only a huginn- label whose owner is gone or whose parent is
// init/launchd, from both the macOS argv form and the Linux process title.
func TestOrphans(t *testing.T) {
	ps := `    1     0 /sbin/launchd
34517     1 tmux -D -L ghycrq-10127-233 -f /dev/null
40001     1 tmux -D -L huginn-golden-99210 -f /dev/null
40002 39990 tmux -D -L huginn-reload-123 -f /dev/null
40003     1 tmux: server (/tmp/tmux-1001/huginn-muninn-tui-77)
40004     1 /opt/homebrew/bin/tmux -D -L huginn-review-7 -f /dev/null
40005     1 tmux new-session -d -s huginn-x
40006     1 tmux: server (/tmp/tmux-1001/default)
40007     1 vim tmux -D -L huginn-not-a-server
40008 40100 tmux -D -L huginn-golden-555 -f /dev/null
40009 40101 /bin/sh -c exec 3<&0 tmux -D -L huginn-golden-555
40010 40102 tmux: server (/tmp/tmux-1001/huginn-zhvkmj-556)
40011 40103 tmux -D -L huginn-odd -f /dev/null
`
	live := map[int]bool{123: true, 556: true}
	got := Orphans(parse(ps), func(pid int) bool { return live[pid] })
	want := []Proc{
		{40001, 1, "huginn-golden-99210"},
		{40003, 1, "huginn-muninn-tui-77"},
		{40004, 1, "huginn-review-7"},
		{40008, 40100, "huginn-golden-555"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orphans\n got %v\nwant %v", got, want)
	}
}

func TestLabelOwner(t *testing.T) {
	l := Label("golden")
	if Owner(l) != os.Getpid() {
		t.Fatalf("Label %q: owner %d, want this process %d", l, Owner(l), os.Getpid())
	}
	// Windows has no signal 0 (and no tmux, so no sweep): liveness is Unix only.
	if runtime.GOOS != "windows" && !running(Owner(l)) {
		t.Fatalf("Label %q: owner %d is not running", l, Owner(l))
	}
	for _, l := range []string{"huginn-odd", "default", "huginn-x-1", "huginn-x-abc"} {
		if Owner(l) != 0 {
			t.Errorf("Owner(%q) = %d, want 0", l, Owner(l))
		}
	}
}

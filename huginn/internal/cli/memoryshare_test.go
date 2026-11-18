package cli

import (
	"strings"
	"testing"

	"github.com/veridit/hug/internal/memshare"
)

// 34r1ft: hug task memory share commits hug.toml [memory] share = true on
// hug/tasks, the setting memshare.Shares reads (Jørgen 18:18, "Clearly A");
// hug task memory local removes it; bare, it says which. Other hug.toml
// lines are kept.
func TestMemoryShareSettingCommand(t *testing.T) {
	t.Parallel()
	a := app(t)
	execute(t, a, "init")
	execute(t, a, "coordinator", "crab")
	if r := execute(t, a, "memory"); r["memory"] != "local" || r["next"] != memshare.Fix {
		t.Fatalf("bare memory on a new board: %v", r)
	}
	if s, err := memshare.Shares(a.Cwd); s || err != nil {
		t.Fatalf("before share: Shares = %v, %v", s, err)
	}
	if r := execute(t, a, "memory", "share"); r["memory"] != "share" || r["changed"] != true || r["next"] != "hug mem sync" {
		t.Fatalf("memory share: %v", r)
	}
	if s, err := memshare.Shares(a.Cwd); !s || err != nil {
		t.Fatalf("after share (committed): Shares = %v, %v", s, err)
	}
	if subject := strings.TrimSpace(git(t, a.Cwd, "log", "-1", "--format=%s", TaskBranch)); subject != "hug task memory share" {
		t.Fatalf("commit subject %q", subject)
	}
	if c := execute(t, a, "coordinator")["coordinator"]; c != "crab" {
		t.Fatalf("the coordinator line was lost: %v", c)
	}
	if r := execute(t, a, "memory"); r["memory"] != "share" {
		t.Fatalf("bare memory after share: %v", r)
	}
	// Twice is no change.
	if r := execute(t, a, "memory", "share"); r["changed"] == true {
		t.Fatalf("memory share twice: %v", r)
	}
	execute(t, a, "memory", "local")
	if s, err := memshare.Shares(a.Cwd); s || err != nil {
		t.Fatalf("after local: Shares = %v, %v", s, err)
	}
	if strings.Contains(git(t, a.Cwd, "show", TaskBranch+":hug.toml"), "[memory]") {
		t.Fatalf("local left an empty [memory] section:\n%s", git(t, a.Cwd, "show", TaskBranch+":hug.toml"))
	}
	if code, out := runText(a, "memory", "yes"); code != 2 || !strings.HasSuffix(out, "fix: hug task memory share\n") {
		t.Fatalf("memory yes: exit %d\n%s", code, out)
	}
}

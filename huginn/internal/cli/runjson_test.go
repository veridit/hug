package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// tfe9nr: a --json reply is read from stdout alone. hug setup --json on a
// changed backlog also prints "hug setup: comparing .backlog/ ..." on
// stderr (backlogcheck.go), a line starting with 'h' like the one that
// broke TestTmuxAutoRefresh. runJSON keeps it out of what is parsed and
// hands it back for the failure message; runText, which merges the two,
// fails to parse ("invalid character 'h' looking for beginning of value").
func TestRunJSONParsesStdoutWhenStderrHasALine(t *testing.T) {
	t.Parallel()
	a := setupBacklogN(t, 3)
	p := filepath.Join(a.Cwd, ".backlog", "tasks", "task-2 - Task-2.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, bytes.Replace(b, []byte("Goal 2."), []byte("Goal 3."), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Second)
	os.Chtimes(p, later, later)
	code, out, serr := runJSON(a, "setup", "--json")
	if !strings.HasPrefix(serr, "hug setup: comparing .backlog/ with the board") {
		t.Fatalf("the fixture printed no stderr line; it tests nothing:\n%s", serr)
	}
	var r struct {
		Setup []SetupStep `json:"setup"`
	}
	if e := json.Unmarshal([]byte(out), &r); e != nil || code != 0 || len(r.Setup) == 0 {
		t.Fatalf("exit %d: %v\nstdout:\n%s\nstderr:\n%s", code, e, out, serr)
	}
}

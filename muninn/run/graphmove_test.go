package run

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/store"
)

// gobGraph has the field names of the graph_state gob from before 4vd1mz.
type gobGraph struct {
	Adj       map[int]map[int]float64
	K         map[int]float64
	Community map[int]int
	SumTot    map[int]float64
	M         float64
}

// asGobStore puts the store back to a graph_state table holding a small
// gob graph, as a hug from before 4vd1mz left it.
func asGobStore(t *testing.T) {
	t.Helper()
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	g := gobGraph{
		Adj:       map[int]map[int]float64{1: {2: 1}, 2: {1: 1}},
		K:         map[int]float64{1: 1, 2: 1},
		Community: map[int]int{1: 1, 2: 1},
		SumTot:    map[int]float64{1: 2},
		M:         1,
	}
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(g); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`DROP VIEW graph_state`, `DELETE FROM graph_vertex`, `DELETE FROM graph_community`,
		`CREATE TABLE graph_state (id INTEGER PRIMARY KEY CHECK (id = 1), state BLOB NOT NULL)`} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(q, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO graph_state(id, state) VALUES(1, ?)`, buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func graphStateIsTable(t *testing.T) bool {
	t.Helper()
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var typ string
	db.QueryRow(`SELECT type FROM sqlite_master WHERE name='graph_state'`).Scan(&typ)
	return typ == "table"
}

// 4vd1mz (crab 04:44): hug mem upgrade previews the concept graph's move
// without changing the exit code or the store (the move needs no upgrade
// step), --apply moves it and says so, hug mem stats shows when, and a
// second upgrade has nothing to do.
func TestMemUpgradeMovesConceptGraph(t *testing.T) {
	memHome(t)
	rememberN(t, 1)
	asGobStore(t)
	out, errOut, code := runCLI(t, "", "upgrade")
	if code != 0 || !strings.Contains(out, "memory: the concept graph is still in its old form") {
		t.Fatalf("preview: exit %d\n%s%s", code, out, errOut)
	}
	if !graphStateIsTable(t) {
		t.Fatal("the preview moved the graph")
	}
	out, errOut, code = runCLI(t, "", "upgrade", "--apply")
	if code != 0 || !strings.Contains(out, "memory: concept graph moved to tables, 2 vertices, ") {
		t.Fatalf("apply: exit %d\n%s%s", code, out, errOut)
	}
	if graphStateIsTable(t) {
		t.Fatal("--apply left graph_state a table")
	}
	logged, _ := os.ReadFile(filepath.Join(config.DataDir(), "hook.log"))
	if !strings.Contains(string(logged), "concept graph moved to tables, 2 vertices, ") {
		t.Fatalf("hook.log in DataDir has no move line:\n%s", logged)
	}
	out, errOut, code = runCLI(t, "", "stats")
	var s struct {
		GraphMoved string `json:"graph_moved"`
	}
	if code != 0 || json.Unmarshal([]byte(out), &s) != nil || !strings.Contains(s.GraphMoved, "concept graph moved to tables, 2 vertices, ") {
		t.Fatalf("stats: exit %d graph_moved %q\n%s%s", code, s.GraphMoved, out, errOut)
	}
	if out, _, code = runCLI(t, "", "upgrade"); code != 0 || strings.TrimSpace(strings.SplitN(out, "\n", 2)[0]) != "memory: nothing to do" {
		t.Fatalf("second run: exit %d\n%s", code, out)
	}
}

// The first hug mem index on a gob store moves the graph and prints the
// line it logged.
func TestMemIndexSaysTheGraphMoved(t *testing.T) {
	memHome(t)
	rememberN(t, 1)
	asGobStore(t)
	out, errOut, code := runCLI(t, "", "index")
	if code != 0 || !strings.Contains(out, "concept graph moved to tables, 2 vertices, ") {
		t.Fatalf("index: exit %d\n%s%s", code, out, errOut)
	}
	if graphStateIsTable(t) {
		t.Fatal("index left graph_state a table")
	}
	if out, _, _ = runCLI(t, "", "index"); strings.Contains(out, "concept graph moved") {
		t.Fatalf("a second index says the graph moved again:\n%s", out)
	}
}

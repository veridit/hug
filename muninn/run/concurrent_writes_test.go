package run

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/veridit/hug/muninn/internal/ingest"
)

// runCLI runs `hug mem args...` in this process and returns stdout, stderr
// and the exit code.
func runCLI(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	dir := t.TempDir()
	outF, _ := os.Create(filepath.Join(dir, "out"))
	errF, _ := os.Create(filepath.Join(dir, "err"))
	inF, _ := os.Create(filepath.Join(dir, "in"))
	inF.WriteString(stdin)
	inF.Seek(0, 0)
	oldOut, oldErr, oldIn := os.Stdout, os.Stderr, os.Stdin
	os.Stdout, os.Stderr, os.Stdin = outF, errF, inF
	code := Main(args)
	os.Stdout, os.Stderr, os.Stdin = oldOut, oldErr, oldIn
	read := func(f *os.File) string {
		f.Seek(0, 0)
		b, _ := io.ReadAll(f)
		f.Close()
		return string(b)
	}
	inF.Close()
	return read(outF), read(errF), code
}

func memHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HUG_HOME", home)
	t.Setenv("HUG_NO_EMBED", "1")
	t.Setenv("JCODE_HOME", filepath.Join(home, "no-jcode"))
	t.Setenv("HOME", filepath.Join(home, "home"))
	t.Chdir(home)
	return home
}

func rememberN(t *testing.T, n int) []string {
	t.Helper()
	var ids []string
	for i := 0; i < n; i++ {
		out, errOut, code := runCLI(t, "", "remember", "me51r2 memory number "+strconv.Itoa(i)+" about quartz "+strings.Repeat("x", i))
		if code != 0 {
			t.Fatalf("remember %d: %s%s", i, out, errOut)
		}
		f := strings.Fields(out)
		if len(f) < 2 || f[0] != "remembered" {
			t.Fatalf("remember %d: %q", i, out)
		}
		ids = append(ids, f[1])
	}
	return ids
}

type memoriesOut struct {
	Memories []struct {
		ID, Kind, Scope, Category, Text string
		Score                           *float64
	} `json:"memories"`
}

func TestListAndSearchJSON(t *testing.T) {
	memHome(t)
	ids := rememberN(t, 3)
	out, errOut, code := runCLI(t, "", "list", "--json")
	if code != 0 {
		t.Fatalf("list --json: %s", errOut)
	}
	var list memoriesOut
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatalf("list --json is not one JSON object: %v\n%s", err, out)
	}
	if len(list.Memories) != 3 || list.Memories[0].ID == "" || !strings.Contains(list.Memories[0].Text, "me51r2 memory") || list.Memories[0].Scope == "" {
		t.Fatalf("list --json: %+v", list)
	}
	if strings.Contains(out, "next:") {
		t.Fatalf("JSON stdout carries a hint:\n%s", out)
	}
	out, errOut, code = runCLI(t, "", "search", "--json", "quartz")
	if code != 0 {
		t.Fatalf("search --json: %s", errOut)
	}
	var found memoriesOut
	if err := json.Unmarshal([]byte(out), &found); err != nil {
		t.Fatalf("search --json is not one JSON object: %v\n%s", err, out)
	}
	if len(found.Memories) == 0 || found.Memories[0].Score == nil || !strings.Contains(strings.Join(ids, " "), found.Memories[0].ID) {
		t.Fatalf("search --json: %s", out)
	}
	// An empty result is still the same shape: {"memories": []}.
	out, _, _ = runCLI(t, "", "search", "--json", "zebrawordnotthere")
	if strings.TrimSpace(out) != `{"memories":[]}` {
		t.Fatalf("empty search --json: %q", out)
	}
}

func TestForgetSeveralIDsAndStdinInOneTransaction(t *testing.T) {
	home := memHome(t)
	ids := rememberN(t, 5)
	// Several ids as arguments.
	out, errOut, code := runCLI(t, "", "forget", ids[0], ids[1])
	if code != 0 || !strings.Contains(out, "forgot "+ids[0]) || !strings.Contains(out, "forgot "+ids[1]) {
		t.Fatalf("forget two: %d %s%s", code, out, errOut)
	}
	// Ids on stdin, one per line, list-style lines and comments allowed.
	out, errOut, code = runCLI(t, "# from hug mem list\n["+ids[2]+"] (memory, global, fact) text\n"+ids[3]+"\n", "forget", "--stdin", "--json")
	if code != 0 {
		t.Fatalf("forget --stdin: %s%s", out, errOut)
	}
	var res struct {
		Forgotten []struct{ ID string } `json:"forgotten"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil || len(res.Forgotten) != 2 || res.Forgotten[0].ID != ids[2] || res.Forgotten[1].ID != ids[3] {
		t.Fatalf("forget --stdin --json: %v %s", err, out)
	}
	// One bad id: nothing is forgotten, the error names it and ends with a
	// fix line, and the good id stays active.
	_, errOut, code = runCLI(t, "", "forget", ids[4], "nosuch1")
	if code == 0 || !strings.Contains(errOut, "nothing forgotten") || !strings.Contains(errOut, "nosuch1") {
		t.Fatalf("bad batch must fail whole: %d %s", code, errOut)
	}
	mems, err := ingest.LoadNative(filepath.Join(home, "data", "memories.json"))
	if err != nil {
		t.Fatal(err)
	}
	active := map[string]bool{}
	for _, m := range mems {
		active[m.SID] = m.Active
	}
	for i, id := range ids {
		if want := i == 4; active[id] != want {
			t.Fatalf("%s active=%v, want %v (all: %v)", id, active[id], want, active)
		}
	}
	// No ids at all is an error with a fix line.
	_, errOut, code = runCLI(t, "", "forget", "--stdin")
	if code == 0 || !strings.Contains(errOut, "fix: hug mem forget") {
		t.Fatalf("empty stdin: %d %s", code, errOut)
	}
}

// me51r2: many agents writing at once is the normal case. 20 processes
// forget at the same moment while another remembers in a loop; every one
// must succeed. Before the write turn this failed with
// "database is locked (5) (SQLITE_BUSY)" once the queue outlasted
// busy_timeout (19 of 20 on a copy of the real store, 2026-11-16).
func TestParallelForgetsAndRememberLoopAllSucceed(t *testing.T) {
	if os.Getenv("HUG_CHILD_ARGS") != "" {
		var args []string
		json.Unmarshal([]byte(os.Getenv("HUG_CHILD_ARGS")), &args)
		os.Exit(Main(args))
	}
	if testing.Short() {
		t.Skip("spawns 30 processes")
	}
	memHome(t)
	const forgets, remembers = 20, 10
	ids := rememberN(t, forgets)
	child := func(args ...string) *exec.Cmd {
		b, _ := json.Marshal(args)
		c := exec.Command(os.Args[0], "-test.run=^TestParallelForgetsAndRememberLoopAllSucceed$")
		c.Env = append(os.Environ(), "HUG_CHILD_ARGS="+string(b), "GOMAXPROCS=1")
		return c
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failures []string
	fail := func(what string, out []byte, err error) {
		mu.Lock()
		defer mu.Unlock()
		failures = append(failures, what+": "+err.Error()+": "+strings.TrimSpace(string(out)))
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < remembers; i++ {
			if out, err := child("remember", "me51r2 concurrent remember "+strconv.Itoa(i)).CombinedOutput(); err != nil {
				fail("remember "+strconv.Itoa(i), out, err)
			}
		}
	}()
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if out, err := child("forget", id).CombinedOutput(); err != nil {
				fail("forget "+id, out, err)
			} else if !strings.Contains(string(out), "forgot "+id) {
				fail("forget "+id, out, os.ErrInvalid)
			}
		}(id)
	}
	wg.Wait()
	if len(failures) > 0 {
		t.Fatalf("%d of %d writes failed:\n%s", len(failures), forgets+remembers, strings.Join(failures, "\n"))
	}
	out, _, _ := runCLI(t, "", "list", "--json", "-n", "1000")
	var list memoriesOut
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Memories) != remembers {
		t.Fatalf("active memories %d, want %d (the remembers only)", len(list.Memories), remembers)
	}
}

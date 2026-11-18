package run

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	huginn "github.com/veridit/hug/huginn/run"
)

// hookDaemon is a jcode daemon at its socket, as far as notify_session
// goes: a live session gets the message appended to its journal as a user
// message (jcode does so before turn_start fires), then done.
type hookDaemon struct {
	home string
	mu   sync.Mutex
	got  []string
}

func newHookDaemon(t *testing.T) *hookDaemon {
	t.Helper()
	// A short path: a Unix socket path is capped near 104 bytes on macOS.
	dir, err := os.MkdirTemp("", "jc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	for _, d := range []string{"sessions", "runtime"} {
		os.MkdirAll(filepath.Join(dir, d), 0o700)
	}
	l, err := net.Listen("unix", filepath.Join(dir, "runtime", "jcode.sock"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	d := &hookDaemon{home: dir}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go d.serve(c)
		}
	}()
	return d
}

func (d *hookDaemon) serve(c net.Conn) {
	defer c.Close()
	sc := bufio.NewScanner(c)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		var raw map[string]any
		json.Unmarshal(sc.Bytes(), &raw)
		id, _ := raw["id"].(float64)
		reply := func(v map[string]any) { b, _ := json.Marshal(v); c.Write(append(b, '\n')) }
		reply(map[string]any{"type": "ack", "id": id})
		if raw["type"] != "notify_session" {
			continue
		}
		sid, _ := raw["session_id"].(string)
		msg, _ := raw["message"].(string)
		line, _ := json.Marshal(map[string]any{"meta": map[string]any{}, "append_messages": []any{map[string]any{"id": "m1", "role": "user", "display_role": "system", "content": []any{map[string]any{"type": "text", "text": msg}}}}})
		jf, _ := os.OpenFile(filepath.Join(d.home, "sessions", sid+".journal.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		jf.Write(append(line, '\n'))
		jf.Close()
		d.mu.Lock()
		d.got = append(d.got, sid+"\x00"+msg)
		d.mu.Unlock()
		reply(map[string]any{"type": "done", "id": id})
	}
}

func (d *hookDaemon) messages() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.got...)
}

// bbxdd7 (crab 15:49): the jcode hook, end to end. A request written while
// crab had no live session waits for crab's name; at crab's next session's
// turn_start the hook hands it over through the daemon, once, and at the
// next tool call the same hook acknowledges it from the transcript.
func TestJcodeHookDeliversPendingAndAcknowledges(t *testing.T) {
	memHome(t)
	jc := newHookDaemon(t)
	for k, v := range map[string]string{"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@e", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@e", "HUG_USER": "jhf", "HUG_MODEL": "human", "JCODE_SESSION_ID": ""} {
		t.Setenv(k, v)
	}
	repo := t.TempDir()
	if out, err := exec.Command("git", "-C", repo, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	exec.Command("git", "-C", repo, "commit", "-q", "--allow-empty", "-m", "init").Run()
	t.Chdir(repo)
	hug := func(args ...string) string {
		t.Helper()
		out, _ := captureSurface(t, func() error {
			if code := huginn.Main(args); code != 0 {
				t.Fatalf("hug task %v: exit %d", args, code)
			}
			return nil
		})
		return out
	}
	hug("init")
	hug("coordinator", "crab")
	id := regexp.MustCompile(`"id":\s*"([a-z0-9]{6})"`).FindStringSubmatch(hug("new", "Tags", "--goal", "g", "--purpose", "p", "--principle", "q", "--accept", "a", "--json"))
	if id == nil {
		t.Fatal("new gave no id")
	}
	if out := hug("request", id[1], "make it per board", "--json"); !strings.Contains(out, "pending, not delivered") {
		t.Fatalf("offline request: %s", out)
	}
	// What the writer leaves for crab's next session (huginn cli
	// TestPendingReachesCoordinatorsNextSession drives that side).
	pend := filepath.Join(jc.home, "hug-inbox", "pending", "crab")
	os.MkdirAll(pend, 0o700)
	it, _ := json.Marshal(map[string]any{"board": repo, "task": id[1], "kind": "request", "n": 1, "to": "crab", "token": "hug " + id[1] + " request 1 old"})
	os.WriteFile(filepath.Join(pend, id[1]+"-request-1.json"), it, 0o600)

	sid := "session_crab_1795000000000_00ab"
	os.WriteFile(filepath.Join(jc.home, "sessions", sid+".json"), []byte(`{"id":"`+sid+`","messages":[]}`), 0o600)
	t.Setenv("JCODE_HOME", jc.home)
	t.Setenv("JCODE_HOOK_SESSION_ID", sid)
	t.Setenv("JCODE_HOOK_CWD", repo)
	event := func(name string) string {
		t.Helper()
		t.Setenv("JCODE_HOOK_EVENT", name)
		t.Setenv("JCODE_HOOK_TOOL_NAME", "bash")
		out, _ := captureSurface(t, func() error { return runHook("jcode", 10*time.Second) })
		return out
	}

	event("turn_start")
	msgs := jc.messages()
	if len(msgs) != 1 || !strings.HasPrefix(msgs[0], sid+"\x00") || !strings.Contains(msgs[0], "make it per board") || !strings.Contains(msgs[0], "hug task resolve "+id[1]+" 1") {
		t.Fatalf("turn_start: daemon got %q", msgs)
	}
	if ents, _ := os.ReadDir(pend); len(ents) != 0 {
		t.Fatalf("pending left after delivery: %v", ents)
	}
	if out := hug("requests", id[1]); !strings.Contains(out, "receipt: pending, delivered to crab") {
		t.Fatalf("after turn_start:\n%s", out)
	}

	if out := event("post_tool_feedback"); !strings.Contains(out, "hug: you received "+id[1]+" request 1") {
		t.Fatalf("post_tool_feedback said %q", out)
	}
	if out := hug("requests", id[1]); !strings.Contains(out, "receipt: acknowledged by crab") {
		t.Fatalf("after the tool call:\n%s", out)
	}

	event("turn_start")
	if n := len(jc.messages()); n != 1 {
		t.Fatalf("delivered again at the next turn: %d messages", n)
	}
}

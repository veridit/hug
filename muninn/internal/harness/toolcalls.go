package harness

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Action is one tool call of an agent session, reduced to what the board
// nudge needs (qawmnb): did it change files in the repository, did it commit,
// and which hug task commands did it run.
type Action struct {
	At   time.Time
	Tool string
	// Paths are the files an edit tool wrote (absolute or as given).
	Paths []string
	// Commit is true for a git commit in a code checkout (not the task
	// branch); Subject is its first message line when it could be read.
	Commit  bool
	Subject string
	// CommitDir is the folder the commit ran in when the command said
	// (cd DIR or git -C DIR), with ~ expanded; empty when unknown.
	CommitDir string
	Hug       []HugCall
}

// HugCall is one hug task command: verb (take, check, note, done, drop,
// new, set, promote, show, ...), the task id and, for check, the item.
type HugCall struct {
	Verb, ID string
	N        int
}

// Activity is what a session did in a window of time.
type Activity struct {
	Actions []Action
	// Model is the model the session runs on, when the transcript says.
	Model string
	ids   []string
}

var (
	hugCmd    = regexp.MustCompile(`\b(?:hug task|huginn|hug-dev task|hugdev task)\s+(take|check|uncheck|note|done|drop|new|set|promote|show|edit|attach|dep|move)\b(?:\s+([a-z0-9]{6})\b)?(?:\s+(\d+)(?:\s|$|;))?`)
	gitCommit = regexp.MustCompile(`\bgit\s+(?:-C\s+(\S+)\s+)?(?:-c\s+\S+\s+)*commit\b([^;&|]*)`)
	commitMsg = regexp.MustCompile(`-[a-zA-Z]*m\s*(?:"((?:[^"\\]|\\.)*)"|'([^']*)'|(\S+))`)
	patchFile = regexp.MustCompile(`(?m)^\*\*\* (?:Update|Add|Delete) File: (.+)$|(?m)^\+\+\+ b/(.+)$`)
	cdDir     = regexp.MustCompile(`(?:^|[;&|(]\s*|\s)cd\s+("[^"]+"|'[^']+'|[^\s;&|)]+)`)
	// Release bookkeeping is not task work: CHANGELOG and version commits.
	bookkeeping = regexp.MustCompile(`^(?i:changelog|release|bump|merge|gofmt)\b|^v\d+\.\d+`)
)

func expandTilde(p string) string {
	home, _ := os.UserHomeDir()
	switch {
	case p == "~" || p == "$HOME":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	case strings.HasPrefix(p, "$HOME/"):
		return filepath.Join(home, p[6:])
	}
	return p
}

// editTools are tool names that write files (jcode, Claude Code, Codex).
var editTools = map[string]bool{
	"edit": true, "write": true, "apply_patch": true, "multiedit": true, "notebookedit": true,
	"str_replace_editor": true, "create_file": true,
}

// ParseToolCall reduces one tool call (name and JSON input) to an Action.
func ParseToolCall(at time.Time, name string, input json.RawMessage) Action {
	a := Action{At: at, Tool: name}
	var in map[string]any
	json.Unmarshal(input, &in)
	lname := strings.ToLower(name)
	str := func(k string) string { s, _ := in[k].(string); return s }
	switch {
	case lname == "batch":
		if calls, ok := in["tool_calls"].([]any); ok {
			for _, c := range calls {
				m, _ := c.(map[string]any)
				tool, _ := m["tool"].(string)
				b, _ := json.Marshal(m)
				sub := ParseToolCall(at, tool, b)
				a.Paths = append(a.Paths, sub.Paths...)
				a.Hug = append(a.Hug, sub.Hug...)
				if sub.Commit {
					a.Commit, a.Subject = true, sub.Subject
				}
			}
		}
	case editTools[lname]:
		if p := str("file_path"); p != "" {
			a.Paths = append(a.Paths, p)
		} else if p := str("path"); p != "" {
			a.Paths = append(a.Paths, p)
		}
		for _, k := range []string{"patch_text", "patch", "input"} {
			for _, m := range patchFile.FindAllStringSubmatch(str(k), -1) {
				p := m[1]
				if p == "" {
					p = m[2]
				}
				a.Paths = append(a.Paths, strings.TrimSpace(p))
			}
		}
	case lname == "bash" || lname == "shell" || lname == "exec_command" || lname == "local_shell":
		cmd := str("command")
		if cmd == "" {
			if xs, ok := in["command"].([]any); ok {
				var parts []string
				for _, x := range xs {
					s, _ := x.(string)
					parts = append(parts, s)
				}
				cmd = strings.Join(parts, " ")
			}
		}
		if cmd == "" {
			cmd = str("cmd")
		}
		a.Hug = HugCalls(cmd)
		for _, ix := range gitCommit.FindAllStringSubmatchIndex(cmd, -1) {
			dir := ""
			if ix[2] >= 0 {
				dir = cmd[ix[2]:ix[3]]
			} else if cds := cdDir.FindAllStringSubmatch(cmd[:ix[0]], -1); len(cds) > 0 {
				dir = cds[len(cds)-1][1]
			}
			dir = expandTilde(strings.Trim(dir, `"'`))
			if strings.Contains(dir, ".hug") || strings.Contains(dir, ".huginn") {
				continue
			}
			a.Commit, a.CommitDir = true, dir
			// The message may hold ; & | inside its quotes: read it from
			// the rest of the command, not the matched clause.
			if s := commitMsg.FindStringSubmatch(cmd[ix[4]:]); s != nil {
				a.Subject = firstLine(s[1] + s[2] + s[3])
			}
		}
	default:
		// MCP task tools: hug's task_take/task_done/task_update under any
		// server prefix (mcp__hug__task_take, hug_task_check, ...).
		if i := strings.LastIndex(lname, "task_"); i >= 0 {
			verb := strings.TrimPrefix(lname[i:], "task_")
			id := str("id")
			if verb == "update" {
				verb = str("action")
			}
			n := 0
			switch v := in["n"].(type) {
			case float64:
				n = int(v)
			case string:
				n, _ = strconv.Atoi(v)
			}
			if verb != "" && (id != "" || verb == "new") && strings.Contains(lname, "hug") || strings.HasPrefix(lname, "task_") {
				a.Hug = append(a.Hug, HugCall{Verb: verb, ID: id, N: n})
			}
		}
	}
	return a
}

// HugCalls finds hug task commands in a shell command line.
func HugCalls(cmd string) []HugCall {
	var out []HugCall
	for _, m := range hugCmd.FindAllStringSubmatch(cmd, -1) {
		n, _ := strconv.Atoi(m[3])
		out = append(out, HugCall{Verb: m[1], ID: m[2], N: n})
	}
	return out
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// EditsIn reports whether a changed a file inside the worktree at root,
// leaving out scratch notes (tmp/), the task board (.hug/) and the
// CHANGELOG: those are not task work. Relative paths count as inside; a
// commit counts when it ran inside root (or nobody said where) and is not
// release bookkeeping.
func (a Action) EditsIn(root string) bool {
	if a.Commit && !bookkeeping.MatchString(a.Subject) && (a.CommitDir == "" || !filepath.IsAbs(a.CommitDir) || inside(root, a.CommitDir)) {
		return true
	}
	for _, p := range a.Paths {
		rel := p
		if filepath.IsAbs(p) {
			if root == "" || !inside(root, p) {
				continue
			}
			rel, _ = filepath.Rel(root, p)
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "tmp/") || strings.HasPrefix(rel, ".hug/") || strings.HasPrefix(rel, ".huginn/") || strings.HasPrefix(rel, ".jcode/") || rel == "CHANGELOG.md" {
			continue
		}
		return true
	}
	return false
}

func inside(root, p string) bool {
	if root == "" {
		return false
	}
	r, err := filepath.Rel(root, p)
	return err == nil && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))
}

// JcodeActivity returns the tool calls a jcode session made after since,
// from the tail of its journal (the newest messages). The model comes from
// the journal's meta.
func JcodeActivity(sessionsDir, sessionID string, since time.Time) (Activity, error) {
	if strings.ContainsAny(sessionID, `/\`) || sessionID == "" {
		return Activity{}, os.ErrInvalid
	}
	base := filepath.Join(sessionsDir, sessionID)
	// The journal holds messages since the last compaction; older ones (and
	// all of them right after a compaction) are at the end of the snapshot.
	act := snapshotTail(base+".json", since)
	f, err := os.Open(base + ".journal.jsonl")
	if err != nil {
		if len(act.Actions) > 0 || !os.IsNotExist(err) {
			return act, nil
		}
		return act, err
	}
	defer f.Close()
	const tail = 2 << 20
	if fi, err := f.Stat(); err == nil && fi.Size() > tail {
		f.Seek(fi.Size()-tail, io.SeekStart)
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	seen := map[string]bool{}
	for _, id := range act.ids {
		seen[id] = true
	}
	for sc.Scan() {
		line := sc.Bytes()
		var entry struct {
			Meta struct {
				Model string `json:"model"`
			} `json:"meta"`
			Append []JcodeStored `json:"append_messages"`
		}
		if !bytes.Contains(line, []byte(`"tool_use"`)) && !bytes.Contains(line, []byte(`"model"`)) {
			continue
		}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		if entry.Meta.Model != "" {
			act.Model = entry.Meta.Model
		}
		for _, m := range entry.Append {
			if m.ID != "" && seen[m.ID] {
				continue
			}
			act.Actions = append(act.Actions, m.Actions(since)...)
		}
	}
	act.ids = nil
	return act, sc.Err()
}

// snapshotTail decodes the messages in the last 4 MB of a jcode snapshot
// (<id>.json) and returns their tool calls after since. The snapshot of a
// long session is tens of MB; its newest messages are at the end.
func snapshotTail(path string, since time.Time) Activity {
	f, err := os.Open(path)
	if err != nil {
		return Activity{}
	}
	defer f.Close()
	const tail = 4 << 20
	if fi, err := f.Stat(); err == nil && fi.Size() > tail {
		f.Seek(fi.Size()-tail, io.SeekStart)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return Activity{}
	}
	var act Activity
	start := []byte(`{"id":"message_`)
	for i := bytes.Index(b, start); i >= 0; {
		dec := json.NewDecoder(bytes.NewReader(b[i:]))
		var m JcodeStored
		if dec.Decode(&m) == nil {
			act.Actions = append(act.Actions, m.Actions(since)...)
			act.ids = append(act.ids, m.ID)
		}
		j := bytes.Index(b[i+len(start):], start)
		if j < 0 {
			break
		}
		i += len(start) + j
	}
	if k := bytes.LastIndex(b, []byte(`"model":"`)); k >= 0 {
		if e := bytes.IndexByte(b[k+9:], '"'); e > 0 {
			act.Model = string(b[k+9 : k+9+e])
		}
	}
	return act
}

// JcodeStored is a jcode message with its tool calls.
type JcodeStored struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Timestamp string `json:"timestamp"`
	Content   []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	} `json:"content"`
}

// Time is the message's timestamp (zero when absent).
func (m JcodeStored) Time() time.Time {
	t, _ := time.Parse(time.RFC3339Nano, m.Timestamp)
	return t
}

// Actions are the message's tool calls after since.
func (m JcodeStored) Actions(since time.Time) []Action {
	if m.Role != "assistant" {
		return nil
	}
	at := m.Time()
	if !at.After(since) {
		return nil
	}
	var out []Action
	for _, c := range m.Content {
		if c.Type == "tool_use" {
			out = append(out, ParseToolCall(at, c.Name, c.Input))
		}
	}
	return out
}

// ClaudeActivity returns the tool calls in a Claude Code transcript
// (transcript_path of the hook payload, JSON lines) after since.
func ClaudeActivity(path string, since time.Time) (Activity, error) {
	f, err := os.Open(path)
	if err != nil {
		return Activity{}, err
	}
	defer f.Close()
	const tail = 2 << 20
	if fi, err := f.Stat(); err == nil && fi.Size() > tail {
		f.Seek(fi.Size()-tail, io.SeekStart)
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
	var act Activity
	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.Contains(line, []byte(`"tool_use"`)) {
			continue
		}
		var e struct {
			Timestamp string `json:"timestamp"`
			Message   struct {
				Role    string `json:"role"`
				Model   string `json:"model"`
				Content []struct {
					Type  string          `json:"type"`
					Name  string          `json:"name"`
					Input json.RawMessage `json:"input"`
				} `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &e) != nil || e.Message.Role != "assistant" {
			continue
		}
		at, _ := time.Parse(time.RFC3339Nano, e.Timestamp)
		if !at.After(since) {
			continue
		}
		if e.Message.Model != "" {
			act.Model = e.Message.Model
		}
		for _, c := range e.Message.Content {
			if c.Type == "tool_use" {
				act.Actions = append(act.Actions, ParseToolCall(at, c.Name, c.Input))
			}
		}
	}
	return act, sc.Err()
}

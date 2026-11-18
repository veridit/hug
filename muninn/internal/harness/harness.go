// Package harness adapts recall to each agent tool's hook protocol.
//
//   - Claude Code and Codex: a synchronous UserPromptSubmit command hook. The
//     hook reads {"prompt","session_id","cwd",...} on stdin and prints
//     {"hookSpecificOutput":{"hookEventName":"UserPromptSubmit","additionalContext":...}}.
//     Context lands before the model answers. (Verified against the Claude Code
//     hooks reference and codex-rs/hooks/src/engine/output_parser.rs.)
//   - jcode: turn_start is a fire-and-forget observer without the prompt, so
//     the adapter reads the turn's user message from the session journal,
//     recalls, and parks the block (park.go). The turn's first successful
//     tool call runs the post_tool_feedback hook, which prints it into that
//     tool result; turn_end drops what no tool call took (z9q42h). Nothing
//     goes into the person's input box.
package harness

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PromptHookInput is the stdin payload of Claude Code / Codex UserPromptSubmit.
type PromptHookInput struct {
	SessionID string `json:"session_id"`
	Prompt    string `json:"prompt"`
	Cwd       string `json:"cwd"`
	Event     string `json:"hook_event_name"`
	// TranscriptPath is the session's JSON-lines transcript (Claude Code);
	// the board nudge reads the session's tool calls from it (qawmnb).
	TranscriptPath string `json:"transcript_path"`
}

// ReadPromptHook parses the hook payload from r.
func ReadPromptHook(r io.Reader) (PromptHookInput, error) {
	var in PromptHookInput
	b, err := io.ReadAll(io.LimitReader(r, 4<<20))
	if err != nil {
		return in, err
	}
	err = json.Unmarshal(b, &in)
	return in, err
}

// PromptHookOutput renders the additionalContext response. An empty block
// yields no output at all (the prompt proceeds unchanged).
func PromptHookOutput(block string) []byte {
	if block == "" {
		return nil
	}
	b, _ := json.Marshal(map[string]any{
		"hookSpecificOutput": map[string]string{
			"hookEventName":     "UserPromptSubmit",
			"additionalContext": block,
		},
	})
	return append(b, '\n')
}

// jcodeMessage is the subset of a jcode message muninn reads.
type jcodeMessage struct {
	ID      string `json:"id"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

// userText returns the typed text of a user message, or "" for tool results
// and harness-generated notices.
func userText(m jcodeMessage) string {
	if m.Role != "user" {
		return ""
	}
	var parts []string
	for _, c := range m.Content {
		if (c.Type == "text" || c.Type == "") && strings.TrimSpace(c.Text) != "" {
			parts = append(parts, c.Text)
		}
	}
	t := strings.TrimSpace(strings.Join(parts, " "))
	if strings.HasPrefix(t, "<system-reminder>") || strings.HasPrefix(t, "You have ") && strings.Contains(t, "incomplete todos") {
		return ""
	}
	return t
}

// JcodeTurn is the turn a jcode session is in: its latest typed user
// message and whether an answer has ended it.
type JcodeTurn struct {
	// ID is the user message's id (a hash of its text when it has none):
	// jcode gives hooks no turn id, so the message is the turn.
	ID   string
	Text string
	// Ended is true once an assistant message with text and no tool call
	// followed the user message: the turn has answered.
	Ended bool
}

// ReadJcodeTurn reads the session's current turn from the journal's last
// 2 MB, and from the snapshot only when that holds no typed user message
// (just after jcode compacts). A recall block earlier hugs delivered as a
// user message is not the user speaking and starts no turn.
func ReadJcodeTurn(sessionsDir, sessionID string) (JcodeTurn, error) {
	var turn JcodeTurn
	if strings.ContainsAny(sessionID, `/\`) || sessionID == "" {
		return turn, os.ErrInvalid
	}
	base := filepath.Join(sessionsDir, sessionID)
	seen := false
	see := func(m jcodeMessage) {
		switch m.Role {
		case "user":
			if t := userText(m); t != "" && !strings.Contains(t, "-- AI memory (muninn):") {
				id := m.ID
				if id == "" {
					h := sha256.Sum256([]byte(t))
					id = "text:" + hex.EncodeToString(h[:8])
				}
				turn, seen = JcodeTurn{ID: id, Text: t}, true
			}
		case "assistant":
			// A final answer: text and no tool call. Tool calls mean the
			// turn goes on.
			text, tool := false, false
			for _, c := range m.Content {
				text = text || (c.Type == "text" && strings.TrimSpace(c.Text) != "")
				tool = tool || c.Type == "tool_use"
			}
			if seen && text && !tool {
				turn.Ended = true
			}
		}
	}
	journal := func() error {
		f, err := os.Open(base + ".journal.jsonl")
		if err != nil {
			return err
		}
		defer f.Close()
		const tail = 2 << 20
		if fi, err := f.Stat(); err == nil && fi.Size() > tail {
			f.Seek(fi.Size()-tail, io.SeekStart)
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 1<<20), 16<<20)
		for sc.Scan() {
			line := sc.Bytes()
			if !bytes.Contains(line, []byte(`"append_messages"`)) {
				continue
			}
			var entry struct {
				Append []jcodeMessage `json:"append_messages"`
			}
			if json.Unmarshal(line, &entry) == nil {
				for _, m := range entry.Append {
					see(m)
				}
			}
		}
		return sc.Err()
	}
	jerr := journal()
	if seen {
		return turn, jerr
	}
	// The snapshot holds the compacted past; the journal appends after it,
	// so read the snapshot first and the journal again on top.
	f, err := os.Open(base + ".json")
	if err != nil {
		if jerr != nil {
			return turn, jerr
		}
		return turn, err
	}
	_, err = streamSnapshot(bufio.NewReaderSize(f, 1<<20), func(c captureStored) {
		see(jcodeMessage{ID: c.ID, Role: c.Role, Content: c.Content})
	})
	f.Close()
	if err != nil {
		return turn, err
	}
	journal()
	return turn, nil
}

// Current reports whether the block parked for turn id may still be shown:
// id is the session's current turn and that turn has not answered. Anything
// else (a newer message, a final answer, an unreadable session) drops it,
// so a block is lost rather than shown in a turn it was not recalled for.
func (t JcodeTurn) Current(id string) bool {
	return id != "" && t.ID == id && !t.Ended
}

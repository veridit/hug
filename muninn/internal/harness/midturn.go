package harness

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Mid-turn recall (x8zzzn, recall.jcode_midturn). jcode runs turn_start only
// when a turn starts. A message the person types while the turn runs (a soft
// interrupt) is appended inside that turn and gets no recall of its own.
//
// jcode names the running turn in every journal line it writes:
// meta.model_usage_turn_id is "<session>:<id of the message that started the
// turn>" (jcode agent/provider.rs begin_model_usage_turn, set before the
// message is saved and before turn_start fires; a soft interrupt keeps it).
// So a typed message whose line carries another message's turn id was typed
// into that turn, and a line carrying a new turn id means a new turn began.
// The park file remembers how far the journal was read (Parked.Offset); each
// tool call reads only what was appended since, so a message is recalled
// once.

// JcodeMsg is a typed user message read from the journal with the turn it
// landed in.
type JcodeMsg struct {
	ID, Text string
	// Person is false for a message jcode wrote with a display role (a
	// swarm DM, a background-task result): not the person typing.
	Person bool
	// Turn is the id of the message that started the jcode turn this one
	// was appended in (meta.model_usage_turn_id); "" for a jcode that does
	// not write it.
	Turn string
	// End is the journal offset just past this message's line.
	End int64
}

// JcodeTail is what a session journal holds after an offset.
type JcodeTail struct {
	Msgs []JcodeMsg
	// Turns are the turn ids the lines read carry, in order, each once.
	Turns []string
	// End is the offset just past the last complete line read: where the
	// next read starts. A line jcode is still writing is left for then.
	End int64
}

// NewTurn returns the first turn id in the tail other than turn, or "": a
// newer turn has begun and turn's block belongs to the past.
func (t JcodeTail) NewTurn(turn string) string {
	for _, id := range t.Turns {
		if id != turn {
			return id
		}
	}
	return ""
}

// Pasted returns the messages typed into turn after the one that started it.
func (t JcodeTail) Pasted(turn string) []JcodeMsg {
	var out []JcodeMsg
	for _, m := range t.Msgs {
		if m.Person && m.Turn == turn && m.ID != turn {
			out = append(out, m)
		}
	}
	return out
}

// jcodeTailMax bounds one read: a journal jcode rewrote (compaction) or a
// park older than this much output is read from its last 2 MB.
const jcodeTailMax = 2 << 20

// ReadJcodeTail reads session sid's journal from offset from. When the
// journal is shorter than from (jcode rewrote it) or more than 2 MB was
// appended since, it reads the last 2 MB instead, from a line start.
func ReadJcodeTail(sessionsDir, sid string, from int64) (JcodeTail, error) {
	var tail JcodeTail
	if strings.ContainsAny(sid, `/\`) || sid == "" || sid == "." || sid == ".." {
		return tail, os.ErrInvalid
	}
	f, err := os.Open(filepath.Join(sessionsDir, sid+".journal.jsonl"))
	if err != nil {
		return tail, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return tail, err
	}
	size := fi.Size()
	skipPartial := false
	if from < 0 || from > size || size-from > jcodeTailMax {
		from = max(0, size-jcodeTailMax)
		skipPartial = from > 0
	}
	if _, err := f.Seek(from, io.SeekStart); err != nil {
		return tail, err
	}
	r := bufio.NewReaderSize(f, 1<<20)
	pos := from
	tail.End = from
	if skipPartial {
		// Started mid-line: drop up to the next line start.
		b, err := r.ReadBytes('\n')
		pos += int64(len(b))
		if err != nil {
			return tail, nil
		}
		tail.End = pos
	}
	prefix := sid + ":"
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			// EOF: a line without its newline is still being written.
			if errors.Is(err, io.EOF) {
				return tail, nil
			}
			return tail, err
		}
		pos += int64(len(line))
		tail.End = pos
		if !bytes.Contains(line, []byte(`"meta"`)) {
			continue
		}
		var entry struct {
			Meta struct {
				Turn string `json:"model_usage_turn_id"`
			} `json:"meta"`
			Append []struct {
				jcodeMessage
				DisplayRole string `json:"display_role"`
			} `json:"append_messages"`
		}
		if json.Unmarshal(line, &entry) != nil {
			continue
		}
		turn := strings.TrimPrefix(entry.Meta.Turn, prefix)
		if turn != "" && (len(tail.Turns) == 0 || tail.Turns[len(tail.Turns)-1] != turn) {
			seen := false
			for _, t := range tail.Turns {
				seen = seen || t == turn
			}
			if !seen {
				tail.Turns = append(tail.Turns, turn)
			}
		}
		for _, m := range entry.Append {
			if m.ID == "" {
				continue
			}
			if t := userText(m.jcodeMessage); t != "" && !strings.Contains(t, "-- AI memory (muninn):") {
				tail.Msgs = append(tail.Msgs, JcodeMsg{ID: m.ID, Text: t, Person: m.DisplayRole == "", Turn: turn, End: pos})
			}
		}
	}
}

// JcodeTurnOf finds message id in the journal's last 2 MB: the jcode turn
// it was appended in (itself, when it started the turn) and the offset past
// it, where mid-turn reading starts. ok is false for a jcode that writes no
// turn ids, or a message not found.
func JcodeTurnOf(sessionsDir, sid, id string) (turn string, end int64, ok bool) {
	tail, err := ReadJcodeTail(sessionsDir, sid, -1)
	if err != nil {
		return "", 0, false
	}
	for _, m := range tail.Msgs {
		if m.ID == id && m.Turn != "" {
			turn, end, ok = m.Turn, m.End, true
		}
	}
	return turn, end, ok
}

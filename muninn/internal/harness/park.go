package harness

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// A jcode recall block waits here between the turn_start hook that computed
// it and the first post_tool_feedback hook of the same turn, which hands it
// to the agent inside a tool result (z9q42h). Nothing hug writes goes into
// the person's input box: jcode transcript --mode send did, while he typed.
//
// The park is one small file per session, <dir>/<session>.json, keyed by the
// id of the user message that started the turn. jcode gives hooks no turn
// id, so the message id is the turn: a park whose message is no longer the
// session's latest is from an earlier turn and is dropped, never shown.

// Parked is a recall block waiting for its turn's first tool result.
type Parked struct {
	// Turn is the id of the user message the block was recalled for.
	Turn  string `json:"turn"`
	Block string `json:"block"`
	// Units are the memories in Block, so delivery can mark them delivered
	// in injections (x8zzzn).
	Units []int64 `json:"units,omitempty"`
	// Midturn (recall.jcode_midturn): Turn is jcode's turn id (the message
	// that started the turn) and Offset is how far the session journal has
	// been read for messages typed into the turn. The park stays, with an
	// empty Block, after its block is delivered, until turn_end or a new
	// turn.
	Midturn bool  `json:"midturn,omitempty"`
	Offset  int64 `json:"offset,omitempty"`
}

// ParkDir is where jcode recall blocks wait, under hug's data folder.
func ParkDir(dataDir string) string { return filepath.Join(dataDir, "jcode-recall") }

func parkPath(dir, session string) (string, error) {
	if session == "" || strings.ContainsAny(session, `/\`) || session == "." || session == ".." {
		return "", fmt.Errorf("bad jcode session id %q", session)
	}
	return filepath.Join(dir, session+".json"), nil
}

// Park stores p for session, replacing an earlier park (a newer turn wins).
func Park(dir, session string, p Parked) error {
	return park(dir, session, p, false)
}

// ParkIfAbsent stores p for session unless a park is already there: a
// mid-turn park put back after a tool call never replaces the park a newer
// turn_start wrote meanwhile. It reports whether p was stored.
func ParkIfAbsent(dir, session string, p Parked) (bool, error) {
	err := park(dir, session, p, true)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	return err == nil, err
}

func park(dir, session string, p Parked, keep bool) error {
	path, err := parkPath(dir, session)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+session+".*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if keep {
		// A hard link fails when the name exists: put back only if absent.
		err := os.Link(tmp.Name(), path)
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return nil
}

// Peek reads session's park without taking it; ok is false when there is
// none.
func Peek(dir, session string) (p Parked, ok bool) {
	path, err := parkPath(dir, session)
	if err != nil {
		return p, false
	}
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &p) != nil {
		return p, false
	}
	return p, true
}

// HasPark reports whether session has a block waiting. It is the whole cost
// of a tool call with nothing parked: one stat, no store, no journal.
func HasPark(dir, session string) bool {
	path, err := parkPath(dir, session)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Claim takes session's park, so that it is delivered at most once even when
// several tool calls of one batch finish together: the rename succeeds for
// exactly one of them. ok is false when there was nothing to take.
func Claim(dir, session string) (p Parked, ok bool, err error) {
	path, err := parkPath(dir, session)
	if err != nil {
		return p, false, err
	}
	claimed := fmt.Sprintf("%s.claimed-%d", path, os.Getpid())
	if err := os.Rename(path, claimed); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return p, false, nil
		}
		return p, false, err
	}
	defer os.Remove(claimed)
	b, err := os.ReadFile(claimed)
	if err != nil {
		return p, false, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return p, false, err
	}
	return p, true, nil
}

package cli

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/task"
)

// catchUpTranslator reads an old-layout commit as layout 3 and merges each
// file it changed with the board's by task.MergeTask (w40cmk check 9, crab
// 04:44).
func catchUpTranslator() gitbranch.Translator {
	same := func(a, b []byte) bool { return (a == nil) == (b == nil) && bytes.Equal(a, b) }
	return gitbranch.Translator{
		Names:   task.Layout3Names,
		Content: task.Layout3Content,
		Layout:  task.LayoutOf,
		Recover: func(p string, base, ours, theirs []byte) bool {
			_, taskFile := task.IDPath(p)
			return !taskFile && !same(ours, base) && !same(theirs, base) && !same(ours, theirs)
		},
		Merge: func(p string, base, ours, theirs []byte, c gitbranch.Replayed) ([]byte, []string, error) {
			if _, ok := task.IDPath(p); !ok {
				// Not a task file (hug.toml, a task's files): the board's
				// when both changed it, else the side that did. The
				// layout in hug.toml is the board's either way.
				switch {
				case same(theirs, base):
					return ours, nil, nil
				default:
					return theirs, nil, nil
				}
			}
			at := c.At
			short := c.Commit
			if len(short) > 12 {
				short = short[:12]
			}
			merged, kept, e := task.MergeTask(p, base, ours, theirs, task.MergeSide{Who: c.Author, At: at, Commit: short, Subject: c.Subject})
			var lines []string
			for _, k := range kept {
				lines = append(lines, keptLine(k))
			}
			return merged, lines, e
		},
	}
}

// keptLine is the one line the run prints for a value both sides changed.
func keptLine(k task.Kept) string {
	one := func(s string) string {
		if s == "" {
			return "(nothing)"
		}
		if strings.Contains(s, "\n") {
			return fmt.Sprintf("%d lines", strings.Count(s, "\n")+1)
		}
		return s
	}
	return fmt.Sprintf("%s %s: kept the board's %s; this clone's %s is in a note on %s", k.ID, k.Field, one(k.Theirs), one(k.Ours), k.ID)
}

// catchUp is the upgrade's catch-up step: the commits it would replay
// (dry) or did, one line each, and one line per value kept in a note.
// Nothing when this clone has nothing to catch up.
func (a *App) catchUp(s *session, dry bool) ([]string, error) {
	t := catchUpTranslator()
	var list []gitbranch.Replayed
	var e error
	if dry {
		list, e = s.r.Pending(t)
	} else {
		list, e = s.r.CatchUp(t)
		s.s.Invalidate()
	}
	if e != nil || len(list) == 0 {
		return nil, e
	}
	replay := 0
	for _, c := range list {
		if !c.Skipped {
			replay++
		}
	}
	head := fmt.Sprintf("the fetched board is translated to layout %d; %d commits of this clone replayed onto it, one by one, each with its own message (nothing pushed until the next sync)", task.Layout, replay)
	if dry {
		head = fmt.Sprintf("would: the fetched board is translated to layout %d; %d commits of this clone are replayed onto it, one by one, each with its own message (nothing pushed until the next sync)", task.Layout, replay)
	}
	lines := []string{head, "append order: explicit times compare actual instants; untimed prose keeps board-before-local order (no chronology inferred)"}
	for _, c := range list {
		short := c.Commit
		if len(short) > 12 {
			short = short[:12]
		}
		if c.Skipped {
			lines = append(lines, fmt.Sprintf("%s %s: not replayed (this clone's own layout move; the remote's stands)", short, c.Subject))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s", short, c.Subject))
		for _, k := range c.Kept {
			lines = append(lines, "  "+k)
		}
	}
	return lines, nil
}

package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// runReflections lists open reflections (tb0dra), or with --all every one;
// --by-instruction groups them by the instruction they name (2k68eg);
// "reflections resolve ID N applied|dismissed" closes one. Dones that came
// without an answer are counted, never listed as reflections.
func (a *App) runReflections(s *session, f flags) (Reply, error) {
	if len(f.args) > 0 && f.args[0] == "resolve" {
		if len(f.args) != 4 {
			return nil, usage("resolve needs ID N applied|dismissed", "hug task reflections resolve ID 1 applied")
		}
		n, err := strconv.Atoi(f.args[2])
		if err != nil || n < 1 {
			return nil, usage("reflection number must be a positive integer", "hug task reflections --all")
		}
		r, e := s.s.ResolveReflection(f.args[1], n, f.args[3])
		if e != nil {
			return nil, e
		}
		if r.Changed {
			if e = s.r.Commit(nil, "hug task reflections resolve "+r.Task.ID+" "+f.args[2]+" "+f.args[3]); e != nil {
				return nil, e
			}
		}
		return Reply{"changed": r.Changed, "next": "hug task reflections"}, nil
	}
	if e := f.count(0); e != nil {
		return nil, e
	}
	rs, unanswered, e := s.s.Reflections(f.has("all"))
	if e != nil {
		return nil, e
	}
	if rs == nil {
		rs = []task.Reflection{}
	}
	if unanswered == nil {
		unanswered = []string{}
	}
	next := "hug task next"
	if first := firstOpen(rs); first != nil {
		next = "hug task reflections resolve " + first.ID + " " + strconv.Itoa(first.N) + " applied (after the instruction change), or dismissed"
	}
	reply := Reply{"unanswered": unanswered, "next": next}
	if f.has("by-instruction") {
		gs := task.GroupReflections(rs)
		if gs == nil {
			gs = []task.ReflectionGroup{}
		}
		if len(gs) > 0 && gs[0].Key != "" {
			if first := firstOpen(gs[0].Reflections); first != nil {
				next = "hug task reflections resolve " + first.ID + " " + strconv.Itoa(first.N) + " applied (after the instruction change), or dismissed"
			}
		}
		reply["groups"], reply["next"] = gs, next
		return reply, nil
	}
	reply["reflections"] = rs
	return reply, nil
}

func firstOpen(rs []task.Reflection) *task.Reflection {
	for i := range rs {
		if rs[i].State == "open" {
			return &rs[i]
		}
	}
	return nil
}

// writeReflection prints one reflection: its line, the instruction it
// quotes, and the text, each indented by pad.
func writeReflection(w io.Writer, r task.Reflection, pad string) {
	who := ""
	if r.Who != "" {
		who = " " + r.Who
	}
	fmt.Fprintf(w, "%s%s %d  %s  %s%s  %s\n", pad, r.ID, r.N, r.State, r.Date, who, clip(r.Title, 60))
	if r.Instruction != "" {
		fmt.Fprintf(w, "%s  instruction: %q\n", pad, r.Instruction)
	}
	fmt.Fprintf(w, "%s  %s\n", pad, r.Text)
}

// reflectionGroupHead is a group's line: what its reflections name and how
// many there are per state, open first.
func reflectionGroupHead(g task.ReflectionGroup) string {
	key := g.Key
	if key == "" {
		key = "no instruction named"
	}
	counts := map[string]int{}
	for _, r := range g.Reflections {
		counts[r.State]++
	}
	var parts []string
	for _, st := range []string{"open", "applied", "dismissed"} {
		if counts[st] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[st], st))
		}
	}
	return key + "  (" + strings.Join(parts, ", ") + ")"
}

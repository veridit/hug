package cli

import (
	"strconv"

	"github.com/veridit/hug/huginn/internal/task"
)

// runReflections lists open reflections (tb0dra), or with --all every one;
// "reflections resolve ID N applied|dismissed" closes one.
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
	rs, e := s.s.Reflections(f.has("all"))
	if e != nil {
		return nil, e
	}
	if rs == nil {
		rs = []task.Reflection{}
	}
	next := "hug task next"
	if len(rs) > 0 && rs[0].State == "open" {
		next = "hug task reflections resolve " + rs[0].ID + " " + strconv.Itoa(rs[0].N) + " applied (after the instruction change), or dismissed"
	}
	return Reply{"reflections": rs, "next": next}, nil
}

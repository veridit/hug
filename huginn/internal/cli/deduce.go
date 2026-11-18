package cli

import (
	"context"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/huginn/internal/task"
)

// Deducing from git (g9dg32): commits that name a ticket in their subject,
// once they are on the main line, and release tags that first contain them,
// are written onto the ticket and reported to its owner. Only certain
// facts: the id in the subject, the commit reachable from main or the tag.
// Never a tick; done only when a release contains a task whose every check
// and child holds (yf39ya), through the ordinary done.
//
// Each clone remembers what it has read in two refs of the code repository
// (refs/hug/seen/main, and refs/hug/seen/tags/<tag> per tag read), so a pass
// reads only what is new. The first pass starts at the current tip: old
// history is not rewritten unless asked (hug task deduce --since REV). A
// note is written once per ticket and commit, whichever clone sees it
// first: the ticket's Notes are checked before writing.

const (
	seenMain = "refs/hug/seen/main"
	seenTags = "refs/hug/seen/tags/"
)

// releaseTag is a release: vX.Y.Z.
var releaseTag = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// codeGit runs git in the code repository at dir.
func codeGit(ctx context.Context, dir string, args ...string) (string, error) {
	out, e := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...).Output()
	return strings.TrimRight(string(out), "\n"), e
}

// mainLine is the ref the main line is read from: the remote's default
// branch when there is one (what landed), else the local branch.
func mainLine(ctx context.Context, dir string) (ref, name string) {
	name = gitbranch.DefaultBranch(dir)
	if name == "" {
		return "", ""
	}
	for _, r := range []string{"refs/remotes/origin/" + name, "refs/heads/" + name} {
		if _, e := codeGit(ctx, dir, "rev-parse", "--verify", "--quiet", r); e == nil {
			return r, name
		}
	}
	return "", ""
}

// commitsIn lists the commits in rng (git log syntax), oldest first.
func commitsIn(ctx context.Context, dir string, rng ...string) ([]task.Commit, error) {
	args := append([]string{"log", "--reverse", "--format=%H%x00%cs%x00%s"}, rng...)
	out, e := codeGit(ctx, dir, args...)
	if e != nil {
		return nil, e
	}
	var cs []task.Commit
	for _, line := range strings.Split(out, "\n") {
		p := strings.SplitN(line, "\x00", 3)
		if len(p) == 3 {
			cs = append(cs, task.Commit{Hash: p[0], Date: p[1], Subject: p[2]})
		}
	}
	return cs, nil
}

// deduction is what one pass found, before it is written.
type deduction struct {
	plan []task.Deduction
	// mark records what was read, once the plan is written.
	mark func() error
}

// planDeductions reads the main line and the release tags new since the
// last pass (or since, when given) and plans the notes they imply.
func planDeductions(ctx context.Context, dir string, s *task.Store, since string) (deduction, error) {
	var none deduction
	ref, name := mainLine(ctx, dir)
	if ref == "" {
		return none, nil
	}
	tip, e := codeGit(ctx, dir, "rev-parse", "--verify", ref)
	if e != nil {
		return none, e
	}
	get := func(id string) *task.Document {
		d, e := s.Get(id)
		if e != nil || d.Task.ID != id {
			return nil // no such ticket (an alias or a prefix is not an id)
		}
		return d
	}
	var plan []task.Deduction
	var marks []func() error
	from := since
	if from == "" {
		from, _ = codeGit(ctx, dir, "rev-parse", "--verify", "--quiet", seenMain)
	}
	if from != "" && from != tip {
		cs, e := commitsIn(ctx, dir, from+".."+tip)
		if e != nil {
			return none, e
		}
		plan = append(plan, task.PlanLanded(cs, name, get)...)
	}
	marks = append(marks, func() error { _, e := codeGit(ctx, dir, "update-ref", seenMain, tip); return e })
	// Tags: each release tag not read yet, against the release before it.
	out, e := codeGit(ctx, dir, "for-each-ref", "--format=%(refname:short)", "refs/tags/v*")
	if e != nil {
		return none, e
	}
	var tags []string
	for _, t := range strings.Fields(out) {
		if releaseTag.MatchString(t) {
			tags = append(tags, t)
		}
	}
	sort.Slice(tags, func(i, j int) bool { return versionLess(tags[i], tags[j]) })
	seen, _ := codeGit(ctx, dir, "for-each-ref", "--format=%(refname)", seenTags)
	read := map[string]bool{}
	for _, r := range strings.Fields(seen) {
		read[strings.TrimPrefix(r, seenTags)] = true
	}
	first := len(read) == 0 && since == ""
	for i, tag := range tags {
		tag := tag
		mark := func() error { _, e := codeGit(ctx, dir, "update-ref", seenTags+tag, "refs/tags/"+tag); return e }
		// A replay (--since) reads every release after since again; the
		// Notes check keeps it to one line per ticket and tag. A release
		// at or before since is marked read, never deduced: a later plain
		// pass must not read it behind the replay's back.
		if since != "" {
			if _, e := codeGit(ctx, dir, "merge-base", "--is-ancestor", "refs/tags/"+tag, since); e == nil {
				if !read[tag] {
					marks = append(marks, mark)
				}
				continue
			}
		} else if read[tag] {
			continue
		}
		marks = append(marks, mark)
		if first {
			continue // the first pass reads no old release
		}
		rng := []string{"refs/tags/" + tag}
		if i > 0 {
			rng = []string{"refs/tags/" + tags[i-1] + "..refs/tags/" + tag}
		}
		cs, e := commitsIn(ctx, dir, rng...)
		if e != nil {
			return none, e
		}
		plan = append(plan, task.PlanShipped(tag, cs, get)...)
	}
	return deduction{plan: plan, mark: func() error {
		for _, m := range marks {
			if e := m(); e != nil {
				return e
			}
		}
		return nil
	}}, nil
}

// versionLess orders vX.Y.Z by number.
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a[1:], "."), strings.Split(b[1:], ".")
	for i := 0; i < 3; i++ {
		if len(pa[i]) != len(pb[i]) {
			return len(pa[i]) < len(pb[i])
		}
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

// deduce runs one deduction pass on an open board: plan, write the notes in
// one commit, then record what was read. The board turn must be held.
// Commit failing leaves the seen refs where they were, so the next pass
// plans the same notes again (and finds those already written).
func (s *session) deduce(ctx context.Context, codeDir, since string) ([]task.Deduction, error) {
	p, e := planDeductions(ctx, codeDir, s.s, since)
	if e != nil || p.mark == nil {
		return nil, e
	}
	written, e := s.s.ApplyDeductions(p.plan)
	if e != nil {
		return written, e
	}
	if len(written) > 0 {
		ids := make([]string, 0, len(written))
		for _, d := range written {
			if !contains(ids, d.ID) {
				ids = append(ids, d.ID)
			}
		}
		if e := s.commit("hug task deduce: " + strings.Join(ids, ", ")); e != nil {
			return written, e
		}
	}
	return written, p.mark()
}

// deduceReply is the reply of hug task deduce: one report line per note.
func deduceReply(ds []task.Deduction) Reply {
	lines := make([]string, len(ds))
	for i, d := range ds {
		lines[i] = d.Report()
	}
	r := Reply{"deductions": lines, "next": "hug task ls"}
	if len(ds) == 0 {
		r["message"] = "nothing new on the main line or in release tags"
	} else {
		r["message"] = strings.Join(lines, "\n")
		r["next"] = "hug task show " + ds[0].ID
	}
	return r
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// handover gives every doing task whose workers have all gone to the
// board's coordinator (yf39ya check 4), in one commit. Only hug task sync
// and the background pass run it (crab 20:09: lint never writes, it runs in
// pre-commit hooks and CI). The board turn must be held.
func (s *session) handover(a *App) ([]task.Handover, error) {
	coordinator, e := task.Coordinator(s.r.Dir)
	if e != nil || coordinator == "" {
		return nil, e
	}
	ts, e := s.s.Tasks()
	if e != nil {
		return nil, e
	}
	people, _ := task.People(s.r.Dir)
	plan := task.PlanHandover(s.liveness(a), ts, coordinator, time.Now(), people...)
	var done []task.Handover
	for _, h := range plan {
		r, e := s.s.HandOver(h)
		if e != nil {
			return done, e
		}
		if r.Changed {
			done = append(done, h)
		}
	}
	if len(done) > 0 {
		ids := make([]string, len(done))
		for i, h := range done {
			ids[i] = h.ID
		}
		if e := s.commit("hug task handover: " + strings.Join(ids, ", ") + " to " + coordinator); e != nil {
			return done, e
		}
	}
	return done, nil
}

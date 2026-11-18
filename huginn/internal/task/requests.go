package task

import (
	"regexp"
	"strconv"
	"strings"
)

// Requests (hbgybn, Jørgen 23:52): a person writes a request on a task in
// their own words, an agent turns it into changes to the ticket and closes
// it with what it did. A request is intent, not a form: it names no field
// or check. It never changes the task's stage, and it stays open until
// someone resolves it. Both texts are kept verbatim.
//
// In the file, one "## Requests" section (before Notes), a block per
// request, its text indented two spaces, the resolve under it indented
// four:
//
//	## Requests
//	- 2026-11-18T00:58 request 1 by jhf:
//	  the text, verbatim
//	  resolved 2026-11-18T01:10 by nautilus:
//	    what changed, verbatim
//
// N numbers the task's requests from 1 and is never reused, so
// "hug task resolve ID N" stays the same request when others close.

// RequestsSection is the heading the requests live under.
const RequestsSection = "Requests"

// Request is one request on a task.
type Request struct {
	N        int         `json:"n"`
	By       string      `json:"by"`
	At       string      `json:"at"`
	Text     string      `json:"text"`
	Resolved *Resolution `json:"resolved,omitempty"`
	// Receipt is how far the request has reached the board's coordinator
	// (gmzd87), read from History.
	Receipt Receipt `json:"receipt"`
	// Targets: one request for several tasks (ec6z6j) is written once, on
	// its home, the first task named; Targets are every task it is for,
	// the home first, each pending or resolved. Nil for a plain request.
	Targets []Target `json:"targets,omitempty"`
	// Home: on another target, the request is a link to the home's
	// request, with no copy of the text.
	Home *RequestRef `json:"home,omitempty"`
	// AllResolved: every target resolved; the request is closed. Read
	// from the home's History ("request N closed: all targets resolved").
	AllResolved bool `json:"all_resolved,omitempty"`
}

// Target is one task a request for several tasks is for.
type Target struct {
	ID       string `json:"id"`
	Resolved bool   `json:"resolved"`
	By       string `json:"by,omitempty"`
}

// RequestRef names a request: the task and its number there.
type RequestRef struct {
	ID string `json:"id"`
	N  int    `json:"n"`
}

// Closed says whether a request for several tasks has every target
// resolved (a plain request: whether it is resolved).
func (r Request) Closed() bool {
	if r.Targets == nil {
		return !r.Open()
	}
	return r.AllResolved
}

// Resolution is how a request was closed: who, when, and what they did.
type Resolution struct {
	By   string `json:"by"`
	At   string `json:"at"`
	Text string `json:"text"`
}

// Open says whether the request waits for an agent.
func (r Request) Open() bool { return r.Resolved == nil }

var (
	requestHead  = regexp.MustCompile(`^- (\S+) request (\d+) by (.+):$`)
	resolvedHead = regexp.MustCompile(`^  resolved (\S+) by (.+):$`)
	// The author of a request for several tasks carries its scope (ec6z6j):
	// "jhf (for a, b, c)" on the home, "jhf (see a request 3)" on a link.
	forScope  = regexp.MustCompile(`^(.+) \(for ([^()]+)\)$`)
	seeScope  = regexp.MustCompile(`^(.+) \(see (\S+) request (\d+)\)$`)
	targetRow = regexp.MustCompile(`^- \S+ request (\d+) target (\S+) resolved by (.+)$`)
	closedRow = regexp.MustCompile(`^- \S+ request (\d+) closed: all targets resolved$`)
)

// scope splits a request head's author into who and the scope.
func scope(r *Request) {
	if m := forScope.FindStringSubmatch(r.By); m != nil {
		r.By = m[1]
		r.Targets = []Target{}
		for _, id := range strings.Split(m[2], ",") {
			if id = strings.TrimSpace(id); id != "" {
				r.Targets = append(r.Targets, Target{ID: id})
			}
		}
		return
	}
	if m := seeScope.FindStringSubmatch(r.By); m != nil {
		n, _ := strconv.Atoi(m[3])
		r.By, r.Home = m[1], &RequestRef{ID: m[2], N: n}
	}
}

// readTargets fills each request's targets from the home's History: the
// home is resolved by its own resolve, a link by its "target ID resolved"
// line, and the request is closed by its "closed" line.
func readTargets(t *Task, history []string) {
	resolved := map[string]string{}
	closed := map[string]bool{}
	for _, l := range history {
		l = strings.TrimRight(l, "\r\n")
		if m := targetRow.FindStringSubmatch(l); m != nil {
			resolved[m[1]+" "+strings.ToLower(m[2])] = m[3]
		}
		if m := closedRow.FindStringSubmatch(l); m != nil {
			closed[m[1]] = true
		}
	}
	for i := range t.Requests {
		r := &t.Requests[i]
		if r.Targets == nil {
			continue
		}
		n := strconv.Itoa(r.N)
		for j := range r.Targets {
			tg := &r.Targets[j]
			if strings.EqualFold(tg.ID, t.ID) {
				if r.Resolved != nil {
					tg.Resolved, tg.By = true, r.Resolved.By
				}
				continue
			}
			if by, ok := resolved[n+" "+strings.ToLower(tg.ID)]; ok {
				tg.Resolved, tg.By = true, by
			}
		}
		r.AllResolved = closed[n]
	}
}

// parseRequests reads the blocks of the Requests section's lines.
func parseRequests(lines []string) []Request {
	var out []Request
	for i := 0; i < len(lines); i++ {
		m := requestHead.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if m == nil {
			continue
		}
		n, _ := strconv.Atoi(m[2])
		r := Request{N: n, At: m[1], By: m[3]}
		var text, account []string
		for i+1 < len(lines) {
			l := strings.TrimRight(lines[i+1], "\r\n")
			if r.Resolved == nil {
				if rm := resolvedHead.FindStringSubmatch(l); rm != nil {
					r.Resolved = &Resolution{At: rm[1], By: rm[2]}
					i++
					continue
				}
			}
			if r.Resolved != nil {
				if rest, ok := strings.CutPrefix(l, "    "); ok {
					account = append(account, rest)
					i++
					continue
				}
				if l == "" || l == "  " {
					// A blank line inside the account, if more of it follows.
					if i+2 < len(lines) && strings.HasPrefix(lines[i+2], "    ") {
						account = append(account, "")
						i++
						continue
					}
				}
				break
			}
			if rest, ok := strings.CutPrefix(l, "  "); ok {
				text = append(text, rest)
				i++
				continue
			}
			break
		}
		r.Text = strings.Join(text, "\n")
		if r.Resolved != nil {
			r.Resolved.Text = strings.Join(account, "\n")
		}
		scope(&r)
		out = append(out, r)
	}
	return out
}

// requestLines are the lines of the Requests section, heading excluded.
func (d *Document) requestLines() []string {
	s, ok := d.sections[RequestsSection]
	if !ok {
		return nil
	}
	return d.lines[s[0]+1 : s[1]]
}

// indent puts prefix before every line of text; an empty line gets the
// prefix too, so the block never breaks.
func indent(text, prefix, nl string) string {
	var b strings.Builder
	for _, l := range strings.Split(text, "\n") {
		b.WriteString(prefix + l + nl)
	}
	return b.String()
}

// cleanText is a request or account as written: line ends unified,
// surrounding blank lines and spaces trimmed, inner text untouched.
func cleanText(text string) string {
	return strings.Trim(strings.ReplaceAll(text, "\r\n", "\n"), " \t\n")
}

// Request stores text as a new open request on ref, by who.
func (s *Store) Request(ref, text, by string) (Result, error) {
	return s.RequestFor([]string{ref}, text, by)
}

// nextRequest is the number d's next request gets: never a reused one.
func nextRequest(d *Document) int {
	n := 1
	for _, r := range d.Task.Requests {
		if r.N >= n {
			n = r.N + 1
		}
	}
	return n
}

// RequestFor stores one request for every task in refs (ec6z6j, Jørgen
// 11:08Z: one request keeps the instruction and the exact selected IDs,
// each selected task links to it, never a duplicate). The text is written
// once, on the first task (its home), with the scope in its head; every
// other task gets a link block with no text. Every ref is resolved and
// the text and author checked before anything is written: an unknown ID
// writes nothing. One ref is a plain request, as before.
func (s *Store) RequestFor(refs []string, text, by string) (Result, error) {
	ts, e := s.Tasks()
	if e != nil {
		return Result{}, e
	}
	var ids, unknown []string
	seen := map[string]bool{}
	for _, ref := range refs {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		t, e := Resolve(ts, ref)
		if e != nil {
			if len(refs) == 1 {
				return Result{}, e
			}
			unknown = append(unknown, strings.TrimSpace(ref))
			continue
		}
		if !seen[strings.ToLower(t.ID)] {
			seen[strings.ToLower(t.ID)] = true
			ids = append(ids, t.ID)
		}
	}
	text, by = cleanText(text), strings.TrimSpace(by)
	list := strings.Join(ids, ",")
	if len(unknown) > 0 {
		fix := "hug task ls"
		if len(ids) > 0 {
			fix = "hug task request " + list + " " + strconv.Quote(oneLineFix(text))
		}
		word := "no task "
		if len(unknown) > 1 {
			word = "no tasks "
		}
		return Result{}, usageErr("", word+strings.Join(unknown, ", ")+": nothing was written; request the known ones, or check the ids", fix)
	}
	if len(ids) == 0 {
		return Result{}, usageErr("", "request needs a task", `hug task request ID "what you want changed"`)
	}
	if text == "" {
		return Result{}, usageErr(ids[0], "request needs the request", "hug task request "+list+` "what you want changed"`)
	}
	if by == "" {
		return Result{}, usageErr(ids[0], "request needs who asks: --as NAME, or HUG_USER", "hug task request "+list+" "+strconv.Quote(oneLineFix(text))+" --as NAME")
	}
	home, _, e := s.load(ids[0])
	if e != nil {
		return Result{}, e
	}
	n := nextRequest(home)
	who := by
	if len(ids) > 1 {
		who = by + " (for " + strings.Join(ids, ", ") + ")"
	}
	block := "- " + stamp() + " request " + strconv.Itoa(n) + " by " + who + ":" + home.newline + indent(text, "  ", home.newline)
	home.appendToSection(RequestsSection, block)
	home.History(stamp() + " request " + strconv.Itoa(n) + " by " + who)
	r, e := s.finish(home, true)
	if e != nil {
		return r, e
	}
	for _, id := range ids[1:] {
		d, _, e := s.load(id)
		if e != nil {
			return r, e
		}
		m := nextRequest(d)
		see := by + " (see " + ids[0] + " request " + strconv.Itoa(n) + ")"
		d.appendToSection(RequestsSection, "- "+stamp()+" request "+strconv.Itoa(m)+" by "+see+":"+d.newline)
		d.History(stamp() + " request " + strconv.Itoa(m) + " by " + see)
		if _, e := s.finish(d, true); e != nil {
			return r, e
		}
	}
	return r, nil
}

// targetResolved records on the home that one target of its request for
// several tasks is resolved, and closes the request when that was the
// last open target (ec6z6j, crab 15:50: in any order; the close line
// lands on the home, after whichever resolve is last).
func (s *Store) targetResolved(home RequestRef, target, by string) error {
	d, _, e := s.load(home.ID)
	if e != nil {
		return e
	}
	var req *Request
	for i := range d.Task.Requests {
		if d.Task.Requests[i].N == home.N && d.Task.Requests[i].Targets != nil {
			req = &d.Task.Requests[i]
		}
	}
	if req == nil {
		return nil // the home no longer holds it: the link's resolve stands
	}
	n := strconv.Itoa(home.N)
	if !strings.EqualFold(target, d.Task.ID) {
		d.History(stamp() + " request " + n + " target " + target + " resolved by " + by)
	}
	open := 0
	for _, tg := range req.Targets {
		if !tg.Resolved && !strings.EqualFold(tg.ID, target) {
			open++
		}
	}
	if open == 0 && !req.AllResolved {
		d.History(stamp() + " request " + n + " closed: all targets resolved")
	}
	s.quiet = true
	defer func() { s.quiet = false }()
	_, e = s.finish(d, true)
	return e
}

// Resolve closes request n on ref with what was done, by who.
func (s *Store) Resolve(ref string, n int, text, by string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	text, by = cleanText(text), strings.TrimSpace(by)
	var found *Request
	var open []string
	var all []string
	for i, r := range d.Task.Requests {
		all = append(all, strconv.Itoa(r.N))
		if r.Open() {
			open = append(open, strconv.Itoa(r.N))
		}
		if r.N == n {
			found = &d.Task.Requests[i]
		}
	}
	if found == nil {
		problem := d.Task.ID + " has no requests"
		if len(all) > 0 {
			problem = d.Task.ID + " has no request " + strconv.Itoa(n) + "; its requests are " + strings.Join(all, ", ") + " (open: " + orNone(open) + ")"
		}
		return Result{}, usageErr(d.Task.ID, problem, "hug task requests "+d.Task.ID)
	}
	if !found.Open() {
		return Result{}, usageErr(d.Task.ID, "request "+strconv.Itoa(n)+" on "+d.Task.ID+" is already resolved, by "+found.Resolved.By+" at "+found.Resolved.At, "hug task show "+d.Task.ID)
	}
	if text == "" {
		return Result{}, usageErr(d.Task.ID, "resolve needs what you changed", "hug task resolve "+d.Task.ID+" "+strconv.Itoa(n)+` "what changed"`)
	}
	if by == "" {
		return Result{}, usageErr(d.Task.ID, "resolve needs who resolves: --as NAME, or HUG_USER", "hug task resolve "+d.Task.ID+" "+strconv.Itoa(n)+" "+strconv.Quote(oneLineFix(text))+" --as NAME")
	}
	// The request's block ends at the next request head or the section end.
	lines := append([]string(nil), d.lines...)
	sec := d.sections[RequestsSection]
	at := -1
	for i := sec[0] + 1; i < sec[1]; i++ {
		m := requestHead.FindStringSubmatch(strings.TrimRight(lines[i], "\r\n"))
		if m != nil && m[2] == strconv.Itoa(n) {
			at = i + 1
			for at < sec[1] && strings.HasPrefix(lines[at], "  ") {
				at++
			}
			break
		}
	}
	if at < 0 {
		return Result{}, &Error{Kind: "broken-file", ID: d.Task.ID, Fix: "hug task lint " + d.Task.ID}
	}
	block := "  resolved " + stamp() + " by " + by + ":" + d.newline + indent(text, "    ", d.newline)
	lines = append(lines[:at], append([]string{block}, lines[at:]...)...)
	d.replace(lines)
	d.History(stamp() + " resolved request " + strconv.Itoa(n) + " by " + by)
	r, e := s.finish(d, true)
	if e != nil {
		return r, e
	}
	// One target of a request for several tasks (ec6z6j): the home keeps
	// the count, and closes the request when this was the last.
	switch {
	case found.Home != nil:
		e = s.targetResolved(*found.Home, d.Task.ID, by)
	case found.Targets != nil:
		e = s.targetResolved(RequestRef{ID: d.Task.ID, N: n}, d.Task.ID, by)
	}
	return r, e
}

func orNone(xs []string) string {
	if len(xs) == 0 {
		return "none"
	}
	return strings.Join(xs, ", ")
}

// OpenRequests are t's requests no one has resolved yet.
func OpenRequests(t Task) []Request {
	var out []Request
	for _, r := range t.Requests {
		if r.Open() {
			out = append(out, r)
		}
	}
	return out
}

// appendToSection adds text at the end of section name, creating the
// section in its canonical place when the file has none.
func (d *Document) appendToSection(name, text string) {
	lines := append([]string(nil), d.lines...)
	s, ok := d.sections[name]
	switch n, before := d.insertAt(name); {
	case !ok && before:
		lines = append(lines[:n], append([]string{"## " + name + d.newline + text + d.newline}, lines[n:]...)...)
	case !ok:
		if len(lines) > 0 && !strings.HasSuffix(lines[len(lines)-1], "\n") {
			lines = append(lines, d.newline)
		}
		lines = append(lines, d.newline+"## "+name+d.newline+text)
	default:
		// After the section's last non-blank line: the blank line that
		// separates it from the next heading stays below.
		end := s[1]
		for end > s[0]+1 && strings.TrimSpace(lines[end-1]) == "" {
			end--
		}
		insert := text
		if end > 0 && !strings.HasSuffix(lines[end-1], "\n") {
			insert = d.newline + insert
		}
		lines = append(lines[:end], append([]string{insert}, lines[end:]...)...)
	}
	d.replace(lines)
}

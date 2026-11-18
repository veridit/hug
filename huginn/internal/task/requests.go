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
)

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
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	text, by = cleanText(text), strings.TrimSpace(by)
	if text == "" {
		return Result{}, usageErr(d.Task.ID, "request needs the request", "hug task request "+d.Task.ID+` "what you want changed"`)
	}
	if by == "" {
		return Result{}, usageErr(d.Task.ID, "request needs who asks: --as NAME, or HUG_USER", "hug task request "+d.Task.ID+" "+strconv.Quote(oneLineFix(text))+" --as NAME")
	}
	n := 1
	for _, r := range d.Task.Requests {
		if r.N >= n {
			n = r.N + 1
		}
	}
	block := "- " + stamp() + " request " + strconv.Itoa(n) + " by " + by + ":" + d.newline + indent(text, "  ", d.newline)
	d.appendToSection(RequestsSection, block)
	d.History(stamp() + " request " + strconv.Itoa(n) + " by " + by)
	return s.finish(d, true)
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
	return s.finish(d, true)
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

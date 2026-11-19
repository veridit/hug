package task

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Merging one task file changed on two sides (w40cmk check 9, crab 04:44;
// ordinary sync and catch-up use the same resolver). base is the file before this
// clone's change, ours after it, theirs the board's. The rule:
//
//  1. Append-only parts merge as the union, in time order where the
//     entries say their time: Notes, History, Depends on lines, new
//     Requests and new acceptance items. Different checks ticked on the
//     two sides both stay ticked. Never a stop.
//  2. A single value changed on both sides (a front-matter field such as
//     state, stage, rank or title; one acceptance item; one request; a
//     section's text): the board's wins, and ours is kept in a note on
//     the task that says who set what, when, in which commit. Each such
//     value is returned in Kept, so the run prints one line for it.
//     Nothing is lost, nothing is silent, nothing stops.
//  3. A file hug cannot read (no front matter, broken YAML), or a task
//     deleted remotely but edited locally, stops with its content retained
//     in the original commit. The latter has no deletion-preservation policy.

// MergeSide says where ours came from, for the note a kept value gets.
type MergeSide struct {
	Who     string // the commit's author
	At      string // its time, 2006-01-02T15:04
	Commit  string // short sha
	Subject string // its message's first line
}

// Kept is one value both sides changed: the board's stays, ours is in a
// note on the task.
type Kept struct {
	ID     string `json:"id"`
	Field  string `json:"field"` // "state", "## Goal", "check 2", "request 1"
	Ours   string `json:"ours"`
	Theirs string `json:"theirs"`
}

// mergeDoc is a task file split for merging: its front matter as keyed
// entries and its body as sections, each kept as raw lines.
type mergeDoc struct {
	present  bool
	newline  string
	fmOrder  []string            // front-matter lines in order: a key, or "\x00" + a raw line that is no key's
	fm       map[string][]string // key -> its raw lines
	preamble []string            // body lines before the first heading
	headings []string
	sections map[string][]string // heading -> its lines, heading line excluded
}

func splitForMerge(path string, raw []byte) (*mergeDoc, error) {
	m := &mergeDoc{fm: map[string][]string{}, sections: map[string][]string{}, newline: "\n"}
	if raw == nil {
		return m, nil
	}
	d := Parse(path, raw)
	if d.frontEnd < 0 || !d.frontmatterValid {
		return nil, fmt.Errorf("hug cannot read this file (no front matter, or front matter that is not YAML)")
	}
	m.present, m.newline = true, d.newline
	starts := map[int]string{}
	for k, n := range d.fields {
		starts[n] = k
	}
	for n := 1; n < d.frontEnd; {
		k, ok := starts[n]
		if !ok {
			m.fmOrder = append(m.fmOrder, "\x00"+d.lines[n])
			n++
			continue
		}
		end := d.fieldEnds[k]
		if end <= n {
			end = n + 1
		}
		m.fmOrder = append(m.fmOrder, k)
		m.fm[k] = append([]string(nil), d.lines[n:end]...)
		n = end
	}
	first := len(d.lines)
	seen := map[string]bool{}
	for _, h := range d.Task.Sections {
		if seen[h] {
			// One heading twice: the sections cannot be told apart.
			return nil, fmt.Errorf("hug cannot read this file (section %s twice)", h)
		}
		seen[h] = true
		s := d.sections[h]
		if s[0] < first {
			first = s[0]
		}
		m.headings = append(m.headings, h)
		m.sections[h] = append([]string(nil), d.lines[s[0]+1:s[1]]...)
	}
	m.preamble = append([]string(nil), d.lines[d.frontEnd+1:first]...)
	return m, nil
}

// MergeTask merges one task file (rule above). base or theirs may be nil
// (the task is new on that side); ours is never nil. kept lists every
// value both sides changed, in file order.
func MergeTask(path string, base, ours, theirs []byte, side MergeSide) ([]byte, []Kept, error) {
	if base != nil && theirs == nil && string(ours) != string(base) {
		id, _ := IDOf(path)
		return nil, nil, fmt.Errorf("task %s was deleted on the board but edited in this commit; its exact local content remains in the original commit; deletion preservation requires an explicit policy", id)
	}
	// A fast path still validates the selected result. Copying malformed
	// bytes unchanged would strand an unreadable task on the new board.
	selected := func(raw []byte) ([]byte, []Kept, error) {
		_, e := splitForMerge(path, raw)
		return raw, nil, e
	}
	switch {
	case theirs == nil && base == nil, string(theirs) == string(base) && theirs != nil:
		return selected(ours)
	case string(ours) == string(base), string(ours) == string(theirs):
		return selected(theirs)
	}
	b, e := splitForMerge(path, base)
	if e != nil {
		return nil, nil, e
	}
	o, e := splitForMerge(path, ours)
	if e != nil {
		return nil, nil, e
	}
	t, e := splitForMerge(path, theirs)
	if e != nil {
		return nil, nil, e
	}
	if !t.present {
		if !b.present {
			return ours, nil, nil
		}
		return nil, nil, nil
	}
	id, _ := IDOf(path)
	var kept []Kept
	keep := func(field string, ov, tv []string, whole bool) {
		kept = append(kept, Kept{ID: id, Field: field, Ours: shown(ov, whole), Theirs: shown(tv, whole)})
	}
	nl := t.newline
	var out []string
	out = append(out, "---"+nl)
	out = append(out, mergeFrontMatter(b, o, t, keep)...)
	out = append(out, "---"+nl)
	pre, c := pick3(b.preamble, o.preamble, t.preamble, b.present, o.present, t.present)
	if c {
		keep("the text above the first section", o.preamble, t.preamble, true)
	}
	out = append(out, pre...)
	for _, h := range mergeHeadings(b.headings, o.headings, t.headings) {
		bs, bok := b.sections[h]
		os_, ook := o.sections[h]
		ts, tok := t.sections[h]
		var lines []string
		switch h {
		case "Notes", "History", "Depends on":
			lines = mergeAppend(h, bs, os_, ts, bok, ook, tok, keep)
		case "Acceptance":
			lines = mergeItems(bs, os_, ts, bok, ook, tok, keep)
		case RequestsSection:
			lines = mergeRequests(bs, os_, ts, bok, ook, tok, keep)
		case QuestionsSection:
			lines = mergeQuestions(bs, os_, ts, bok, ook, tok, keep)
		default:
			var c bool
			var present bool
			lines, present, c = pick3p(bs, os_, ts, bok, ook, tok)
			if c {
				keep("## "+h, os_, ts, true)
			}
			if !present {
				continue
			}
		}
		if lines == nil && !(bok || ook || tok) {
			continue
		}
		out = append(out, "## "+h+nl)
		out = append(out, lines...)
	}
	merged := []byte(strings.Join(out, ""))
	if len(kept) == 0 {
		return merged, nil, nil
	}
	d := Parse(path, merged)
	for _, k := range kept {
		note := keptNote(k, side)
		if !strings.Contains(d.Task.Texts.Notes, note) {
			d.Note(note)
		}
	}
	if e := strayToHistory(d, kept, side); e != nil {
		return nil, nil, e
	}
	return d.Bytes(), kept, nil
}

// strayToHistory keeps the board's state whole (8zwy6r): when state was
// changed on both sides and the board's kept, the local move's node
// fields (dropped_reason and dropped_at of a drop) belong to the state
// that lost. Left in the front matter they lint broken
// [stray-node-field]; they go to ## History, as a move puts them, at the
// local commit's time so both clones write the same bytes.
func strayToHistory(d *Document, kept []Kept, side MergeSide) error {
	state := false
	for _, k := range kept {
		if k.Field == "state" {
			state = true
		}
	}
	if !state {
		return nil
	}
	owners := NodeFields()
	var moved []string
	for _, k := range sortedKeys(d.Task.Fields) {
		if o, node := owners[k]; node && !fieldHolds(d.Task, k, o) {
			moved = append(moved, k+": "+d.Task.Fields[k])
			if e := d.RemoveField(k); e != nil {
				return e
			}
		}
	}
	if len(moved) > 0 {
		d.History(side.At + " sync kept the board's state " + d.Task.State + "; this clone's " + strings.Join(moved, ", "))
	}
	return nil
}

// keptNote is the note that keeps ours: who set what here, when, in which
// commit, what the board had, and that the board's was kept (crab 04:44:
// "kestrel set state doing here at T; the board had done; kept done").
func keptNote(k Kept, s MergeSide) string {
	where := fmt.Sprintf("%s (commit %s %q)", s.At, s.Commit, s.Subject)
	if strings.HasPrefix(k.Field, "competing resolution for request ") {
		return fmt.Sprintf("Competing request resolution retained: %s on task %s. The board's resolution remains canonical. Local replay by %s at %s. Local request and resolution (verbatim):\n%s", k.Field, k.ID, s.Who, where, quoteBlock(k.Ours))
	}
	if strings.Contains(k.Ours, "\n") || strings.Contains(k.Theirs, "\n") {
		return fmt.Sprintf("%s set %s here at %s; the board had changed it too; kept the board's. This clone's text was:\n%s", s.Who, k.Field, where, quoteBlock(k.Ours))
	}
	return fmt.Sprintf("%s set %s %s here at %s; the board had %s; kept %s", s.Who, k.Field, orNothing(k.Ours), where, orNothing(k.Theirs), orNothing(k.Theirs))
}

func orNothing(v string) string {
	if v == "" {
		return "(nothing)"
	}
	return v
}

func quoteBlock(s string) string {
	var b strings.Builder
	for i, l := range strings.Split(s, "\n") {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("> " + l)
	}
	return b.String()
}

// shown is a value as a note and the run print it: a front-matter value
// without its key and quotes, a section's text as it is.
func shown(lines []string, whole bool) string {
	s := strings.TrimRight(strings.ReplaceAll(strings.Join(lines, ""), "\r\n", "\n"), "\n")
	if whole {
		return strings.Trim(s, "\n")
	}
	if _, v, ok := strings.Cut(s, ":"); ok && !strings.Contains(s, "\n") {
		return scalar(v)
	}
	return s
}

// pick3 is the three-way choice for one value; conflict when both sides
// changed it differently (the board's then wins).
func pick3(b, o, t []string, bok, ook, tok bool) ([]string, bool) {
	v, _, c := pick3p(b, o, t, bok, ook, tok)
	return v, c
}

func pick3p(b, o, t []string, bok, ook, tok bool) (v []string, present, conflict bool) {
	same := func(x, y []string, xok, yok bool) bool { return xok == yok && equalText(x, y) }
	switch {
	case same(o, b, ook, bok):
		return t, tok, false
	case same(t, b, tok, bok):
		return o, ook, false
	case same(o, t, ook, tok):
		return t, tok, false
	}
	return t, tok, true
}

func equalText(x, y []string) bool {
	norm := func(l []string) string {
		return strings.TrimRight(strings.ReplaceAll(strings.Join(l, ""), "\r\n", "\n"), "\n ")
	}
	return norm(x) == norm(y)
}

// sameRelease reports both sides closed by one release tag: done_release
// set, and the same, on each.
func sameRelease(o, t *mergeDoc) bool {
	ov, ook := o.fm["done_release"]
	tv, tok := t.fm["done_release"]
	return ook && tok && equalText(ov, tv)
}

// mergeFrontMatter merges key by key in the board's order; a key only
// ours added goes after the key it follows in ours. updated: is the later
// of the two, never a kept value.
func mergeFrontMatter(b, o, t *mergeDoc, keep func(string, []string, []string, bool)) []string {
	value := func(k string) ([]string, bool) {
		bv, bok := b.fm[k]
		ov, ook := o.fm[k]
		tv, tok := t.fm[k]
		if k == "updated" && ook && tok {
			if scalar(strings.SplitN(strings.Join(ov, ""), ":", 2)[1]) > scalar(strings.SplitN(strings.Join(tv, ""), ":", 2)[1]) {
				return ov, true
			}
			return tv, true
		}
		// One release close on two clones (j0zvee): each stamped done_at
		// at its own minute. The same tag closed it on both, so it is one
		// close: the board's minute, and nothing was overruled.
		if k == "done_at" && ook && tok && !bok && sameRelease(o, t) {
			return tv, true
		}
		v, present, c := pick3p(bv, ov, tv, bok, ook, tok)
		if c {
			keep(k, ov, tv, false)
		}
		return v, present
	}
	added := map[string][]string{} // key in the result -> ours-only keys that follow it
	var lead []string
	prev := ""
	for _, k := range o.fmOrder {
		if strings.HasPrefix(k, "\x00") {
			continue
		}
		_, inT := t.fm[k]
		_, inB := b.fm[k]
		if !inT && !inB {
			if prev == "" {
				lead = append(lead, k)
			} else {
				added[prev] = append(added[prev], k)
			}
			continue
		}
		prev = k
	}
	var out []string
	emit := func(k string) {
		if v, ok := value(k); ok {
			out = append(out, v...)
		}
		for _, a := range added[k] {
			out = append(out, o.fm[a]...)
		}
	}
	for _, k := range lead {
		out = append(out, o.fm[k]...)
	}
	order := t.fmOrder
	for _, k := range order {
		if strings.HasPrefix(k, "\x00") {
			out = append(out, k[1:])
			continue
		}
		emit(k)
	}
	// Keys the board removed and ours changed, or that only base and ours
	// have (ours' place): after the board's keys.
	for _, k := range o.fmOrder {
		if strings.HasPrefix(k, "\x00") {
			continue
		}
		if _, inT := t.fm[k]; inT {
			continue
		}
		if _, inB := b.fm[k]; !inB {
			continue // added: emitted above
		}
		if v, ok := value(k); ok {
			out = append(out, v...)
		}
		// A new key must survive even when its anchor was removed.
		for _, a := range added[k] {
			out = append(out, o.fm[a]...)
		}
	}
	// Comments outside a key are still user content. Keep new local raw
	// lines even when both sides changed keyed fields around them.
	rawSeen := map[string]bool{}
	for _, order := range [][]string{b.fmOrder, t.fmOrder} {
		for _, line := range order {
			if strings.HasPrefix(line, "\x00") {
				rawSeen[line] = true
			}
		}
	}
	for _, line := range o.fmOrder {
		if strings.HasPrefix(line, "\x00") && !rawSeen[line] {
			out = append(out, line[1:])
			rawSeen[line] = true
		}
	}
	return out
}

// mergeHeadings is the board's sections in its order, with a section
// only ours has put in its canonical place.
func mergeHeadings(b, o, t []string) []string {
	out := append([]string(nil), t...)
	has := map[string]bool{}
	for _, h := range t {
		has[h] = true
	}
	for _, h := range o {
		if has[h] {
			continue
		}
		has[h] = true
		at := len(out)
		for i, x := range out {
			if sectionRank(x) > sectionRank(h) {
				at = i
				break
			}
		}
		out = append(out[:at], append([]string{h}, out[at:]...)...)
	}
	// A section only base had (both sides removed it) is not written.
	return out
}

var entryTime = regexp.MustCompile(`^(?:- )?(\d{4}-\d\d-\d\d(?:T\d\d:\d\d(?::\d\d(?:\.\d+)?)?(?:Z|[+-]\d\d:\d\d)?)?)\b`)

// mergeAppend is the union of both append-only sections (rule 1). Compare
// entries, not a common text prefix: an earlier replay may have inserted
// remote entries between the base's entries. Timed entries are ordered
// chronologically, with untimed entries retaining their board-first order.
func mergeAppend(h string, b, o, t []string, bok, ook, tok bool, keep func(string, []string, []string, bool)) []string {
	if !tok {
		return o
	}
	if !ook {
		return t
	}
	ob, _ := trimBlank(o)
	tb, tail := trimBlank(t)
	bb, _ := trimBlank(b)
	// This commit's actual append is reliable even when the remote no
	// longer begins with base. hug task note also permits adjacent notes
	// without blank separators, so unioning entire paragraphs would repeat
	// an earlier local note together with its newly appended continuation.
	if hasPrefix(ob, bb) {
		ob = ob[len(bb):]
	}
	// An unseparated note may already be embedded in the board's larger
	// paragraph. Match complete appended line sequences, not substrings.
	if h == "Notes" {
		for i := 0; i < len(tb); i++ {
			if len(ob) > 0 && hasPrefix(tb[i:], ob) {
				ob = nil
				break
			}
		}
	}
	var entries [][]string
	seen := map[string]bool{}
	for _, lines := range [][]string{tb, ob} {
		for _, entry := range blocks(h, lines) {
			key := strings.TrimSpace(strings.ReplaceAll(strings.Join(entry, ""), "\r\n", "\n"))
			if h == "History" {
				if k, ok := releaseCloseKey(key); ok {
					key = k
				}
			}
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			entries = append(entries, entry)
		}
	}
	// Sort timed entries only, leaving untimed entries in their original
	// slots. Comparing a timed entry with an untimed one as equal would
	// make the sort's ordering non-transitive.
	var positions []int
	var timed [][]string
	for i, entry := range entries {
		if blockTime(entry) != "" {
			positions = append(positions, i)
			timed = append(timed, entry)
		}
	}
	sort.SliceStable(timed, func(i, j int) bool { return blockTime(timed[i]) < blockTime(timed[j]) })
	for i, at := range positions {
		entries[at] = timed[i]
	}
	var out []string
	for _, entry := range entries {
		out = append(out, entry...)
	}
	return append(out, tail...)
}

// releaseCloseKey is the merge key of a History line a release close wrote
// (j0zvee): its transition and its reason, without the minute. Two clones
// that deduce the same tag each close the task by it at their own minute;
// that is one event, so the board's line is kept and the other dropped,
// whatever minute either stamped. Any other line keeps its time in its
// key: two resumes a minute apart on two clones are two events.
func releaseCloseKey(entry string) (string, bool) {
	line, _, _ := strings.Cut(entry, "\n")
	m := entryTime.FindStringSubmatchIndex(line)
	if m == nil {
		return "", false
	}
	rest := strings.TrimSpace(line[m[1]:])
	move, why, ok := strings.Cut(rest, " (")
	if !ok || !strings.Contains(move, " -> ") {
		return "", false
	}
	why, _, ok = strings.Cut(why, ")")
	if !ok || !strings.HasPrefix(why, "done, by release ") {
		return "", false
	}
	return "release close\x00" + move + "\x00" + why, true
}

// blocks splits appended lines into entries: a History or Depends on line
// starting "- " starts one; a Notes entry is its blank separator line and
// its text, so it starts at a blank line after text.
func blocks(h string, lines []string) [][]string {
	var out [][]string
	prevBlank := false
	for _, l := range lines {
		blank := strings.TrimSpace(l) == ""
		start := len(out) == 0
		switch h {
		case "Notes":
			start = start || (blank && !prevBlank)
		default:
			start = start || strings.HasPrefix(l, "- ")
		}
		if start {
			out = append(out, nil)
		}
		out[len(out)-1] = append(out[len(out)-1], l)
		prevBlank = blank
	}
	return out
}

func blockTime(b []string) string {
	for _, l := range b {
		if strings.TrimSpace(l) == "" {
			continue
		}
		if m := entryTime.FindStringSubmatch(l); m != nil {
			for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04Z07:00", "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
				if at, e := time.Parse(layout, m[1]); e == nil {
					return at.UTC().Format("2006-01-02T15:04:05.000000000Z")
				}
			}
		}
		return ""
	}
	return ""
}

func trimBlank(l []string) (body, tail []string) {
	n := len(l)
	for n > 0 && strings.TrimSpace(l[n-1]) == "" {
		n--
	}
	return l[:n], l[n:]
}

func hasPrefix(l, p []string) bool {
	if len(p) > len(l) {
		return false
	}
	for i := range p {
		if strings.TrimRight(l[i], "\r\n") != strings.TrimRight(p[i], "\r\n") {
			return false
		}
	}
	return true
}

// keyed splits a section into its lines before the first entry and its
// entries, each starting at a line start says starts one.
func keyed(lines []string, start func(string) bool) (pre []string, entries [][]string) {
	for _, l := range lines {
		if start(strings.TrimRight(l, "\r\n")) {
			entries = append(entries, []string{l})
			continue
		}
		if len(entries) == 0 {
			pre = append(pre, l)
		} else {
			entries[len(entries)-1] = append(entries[len(entries)-1], l)
		}
	}
	return pre, entries
}

func isItem(l string) bool {
	t := strings.TrimSpace(l)
	return strings.HasPrefix(t, "- [ ] ") || strings.HasPrefix(t, "- [x] ") || strings.HasPrefix(t, "- [X] ")
}

// mergeItems merges acceptance items by number: each item a single value
// (a check ticked on one side is that side's change), new items on both
// sides appended, the board's first.
func mergeItems(b, o, t []string, bok, ook, tok bool, keep func(string, []string, []string, bool)) []string {
	if !tok {
		return o
	}
	if !ook {
		return t
	}
	bp, be := keyed(b, isItem)
	op, oe := keyed(o, isItem)
	tp, te := keyed(t, isItem)
	if len(oe) < len(be) || len(te) < len(be) {
		// An item removed (hug strikes, never removes): one value.
		v, c := pick3(b, o, t, bok, ook, tok)
		if c {
			keep("## Acceptance", o, t, true)
		}
		return v
	}
	out, c := pick3(bp, op, tp, true, true, true)
	if c {
		keep("the text above the acceptance items", op, tp, true)
	}
	for n := range be {
		v, c := pick3(be[n], oe[n], te[n], true, true, true)
		if c {
			keep(fmt.Sprintf("check %d", n+1), itemOnly(oe[n]), itemOnly(te[n]), true)
		}
		out = append(out, itemOnly(v)...)
	}
	// New items: the board's, then ours. The blank lines that end the
	// section are the board's (they sit on its last item).
	for _, v := range te[len(be):] {
		out = append(out, itemOnly(v)...)
	}
	for _, v := range oe[len(be):] {
		out = append(out, itemOnly(v)...)
	}
	var tail []string
	switch {
	case len(te) > 0:
		_, tail = trimBlank(te[len(te)-1])
	case len(oe) > 0:
		_, tail = trimBlank(oe[len(oe)-1])
	}
	return append(out, tail...)
}

func itemOnly(l []string) []string { body, _ := trimBlank(l); return body }

// mergeRequests follows a request by its original time, author and text,
// not its display number. Earlier replay may have renumbered a local
// request after a collision, and a later resolve must follow that request.
func mergeRequests(b, o, t []string, bok, ook, tok bool, keep func(string, []string, []string, bool)) []string {
	return mergeNumbered(numberedKind{noun: "request", head: requestHead, closed: resolvedHead, renumber: func(header string, n int) string {
		return requestHead.ReplaceAllString(header, "- ${1} request "+strconv.Itoa(n)+" by ${3}:")
	}}, b, o, t, bok, ook, tok, keep)
}

// mergeQuestions merges ## Questions (q2adr8) as requests merge: a
// question asked on both sides keeps both, the later one renumbered past
// the other's; an answer under it on one side is kept.
func mergeQuestions(b, o, t []string, bok, ook, tok bool, keep func(string, []string, []string, bool)) []string {
	return mergeNumbered(numberedKind{noun: "question", head: questionHead, closed: closedUnder, renumber: func(header string, n int) string {
		m := questionHead.FindStringSubmatchIndex(header)
		if m == nil {
			return header
		}
		return header[:m[4]] + strconv.Itoa(n) + header[m[5]:]
	}}, b, o, t, bok, ook, tok, keep)
}

// numberedKind is a section of numbered blocks: its head line (number in
// group 2), the line that closes one (resolved, answered), and how a head
// gets a new number.
type numberedKind struct {
	noun     string
	head     *regexp.Regexp
	closed   *regexp.Regexp
	renumber func(header string, n int) string
}

func mergeNumbered(k numberedKind, b, o, t []string, bok, ook, tok bool, keep func(string, []string, []string, bool)) []string {
	if !tok {
		return o
	}
	if !ook {
		return t
	}
	start := func(l string) bool { return k.head.MatchString(l) }
	num := func(e []string) int {
		m := k.head.FindStringSubmatch(strings.TrimRight(e[0], "\r\n"))
		n, _ := strconv.Atoi(m[2])
		return n
	}
	bp, be := keyed(b, start)
	op, oe := keyed(o, start)
	tp, te := keyed(t, start)
	renumber := func(e []string, n int) []string {
		if len(e) == 0 {
			return e
		}
		out := append([]string(nil), e...)
		header := strings.TrimRight(out[0], "\r\n")
		out[0] = k.renumber(header, n) + strings.TrimPrefix(out[0], header)
		return out
	}
	key := func(e []string) string {
		core := renumber(e, 0)
		for i, l := range core {
			if k.closed.MatchString(strings.TrimRight(l, "\r\n")) {
				core = core[:i]
				break
			}
		}
		return strings.TrimSpace(strings.ReplaceAll(strings.Join(core, ""), "\r\n", "\n"))
	}
	byKey := func(es [][]string) map[string][]string {
		m := map[string][]string{}
		for _, e := range es {
			m[key(e)] = e
		}
		return m
	}
	bm, om, tm := byKey(be), byKey(oe), byKey(te)
	out, c := pick3(bp, op, tp, bok, ook, tok)
	if c {
		keep("the text above the "+k.noun+"s", op, tp, true)
	}
	max := 0
	used := map[int]bool{}
	for _, e := range te {
		n := num(e)
		used[n] = true
		if n > max {
			max = n
		}
	}
	var entries [][]string
	for _, e := range te {
		n := num(e)
		key0 := key(e)
		bv, bok := bm[key0]
		ov, ook := om[key0]
		if !bok && !ook {
			entries = append(entries, itemOnly(e))
			continue
		}
		if !bok {
			// The same request was independently added or is already on
			// the board. Its initial text is the base for any resolution.
			bv = e
			for i, l := range bv {
				if k.closed.MatchString(strings.TrimRight(l, "\r\n")) {
					bv = bv[:i]
					break
				}
			}
			bok = true
		}
		v, _, c := pick3p(renumber(bv, n), renumber(ov, n), e, bok, ook, true)
		if c {
			field := fmt.Sprintf("%s %d", k.noun, n)
			resolved := func(lines []string) bool {
				for _, line := range lines {
					if k.closed.MatchString(strings.TrimRight(line, "\r\n")) {
						return true
					}
				}
				return false
			}
			if resolved(ov) && resolved(e) {
				field = fmt.Sprintf("competing resolution for %s %d (identity %x)", k.noun, n, sha256.Sum256([]byte(key0)))
			}
			keep(field, itemOnly(ov), itemOnly(e), true)
		}
		entries = append(entries, itemOnly(v))
	}
	for _, e := range oe {
		if _, inT := tm[key(e)]; inT {
			continue
		}
		n := num(e)
		if used[n] {
			max++
			n = max
		} else if n > max {
			max = n
		}
		used[n] = true
		entries = append(entries, itemOnly(renumber(e, n)))
	}
	sort.SliceStable(entries, func(i, j int) bool { return blockTime(entries[i]) < blockTime(entries[j]) })
	for _, e := range entries {
		out = append(out, e...)
	}
	var tail []string
	if len(te) > 0 {
		_, tail = trimBlank(te[len(te)-1])
	} else if len(oe) > 0 {
		_, tail = trimBlank(oe[len(oe)-1])
	}
	return append(out, tail...)
}

package task

import (
	"bytes"
	"regexp"
)

// Record and live text (7j706s). A task file holds two kinds of text: live
// text says what is true now (Goal, Purpose, Details, Acceptance, Notes, an
// open question or request), and record text quotes the past: ## History,
// an answered or withdrawn question, a resolved request. A link in live
// text must resolve; a citation in record text is what someone wrote then
// (p87ngv's answered question 5 proposed .hug/tasks/p/p87ngv/history.md,
// and jhf chose history.sdd), so lint does not check it.

// closedMark is the line under a question or request that closes it: its
// answer, its withdrawal or its resolve.
var closedMark = regexp.MustCompile(`^  (?:answered \S+ by .+:|withdrawn \S+ \(.*\)|resolved \S+ by .+:)\s*$`)

// recordSpans are the byte ranges [start, end) of doc that are record text:
// the ## History section, and each closed block of ## Questions and
// ## Requests (from its "- " head to the next head or section). Headings
// inside a fenced block are text, as the parser reads them.
func recordSpans(doc []byte) [][2]int {
	var spans [][2]int
	section := ""
	fence := ""
	block, closed := -1, false
	endBlock := func(at int) {
		if block >= 0 && closed {
			spans = append(spans, [2]int{block, at})
		}
		block, closed = -1, false
	}
	historyStart := -1
	for pos := 0; pos < len(doc); {
		end := bytes.IndexByte(doc[pos:], '\n')
		next := len(doc)
		if end >= 0 {
			next = pos + end + 1
		}
		line := bytes.TrimRight(doc[pos:next], "\r\n")
		trimmed := bytes.TrimLeft(line, " ")
		if len(line)-len(trimmed) <= 3 && (bytes.HasPrefix(trimmed, []byte("```")) || bytes.HasPrefix(trimmed, []byte("~~~"))) {
			if fence == "" {
				n := 0
				for n < len(trimmed) && trimmed[n] == trimmed[0] {
					n++
				}
				fence = string(trimmed[:n])
			} else if bytes.HasPrefix(trimmed, []byte(fence)) && len(bytes.TrimSpace(trimmed[len(fence):])) == 0 {
				fence = ""
			}
		} else if fence == "" && bytes.HasPrefix(line, []byte("## ")) {
			endBlock(pos)
			if historyStart >= 0 {
				spans = append(spans, [2]int{historyStart, pos})
				historyStart = -1
			}
			section = string(bytes.TrimSpace(line[3:]))
			if section == "History" {
				historyStart = pos
			}
		} else if fence == "" && (section == QuestionsSection || section == RequestsSection) {
			if bytes.HasPrefix(line, []byte("- ")) {
				endBlock(pos)
				block = pos
			} else if block >= 0 && closedMark.Match(line) {
				closed = true
			}
		}
		pos = next
	}
	endBlock(len(doc))
	if historyStart >= 0 {
		spans = append(spans, [2]int{historyStart, len(doc)})
	}
	return spans
}

// inSpans says whether pos falls in one of spans.
func inSpans(spans [][2]int, pos int) bool {
	for _, s := range spans {
		if pos >= s[0] && pos < s[1] {
			return true
		}
	}
	return false
}

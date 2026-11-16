package cli

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// Lint output stays readable on big boards: past lintBound findings, plain
// lint prints the summary and the first lintShown findings, then how to see
// all (6neax1: 2,478 findings, 318 KB, came back as nothing in an agent's
// tool panel).
const (
	lintBound = 30
	lintShown = 10
)

// LintSummary is the summary object of a lint reply. Total and Codes are
// in JSON; the rest only steer the text output.
type LintSummary struct {
	Total int              `json:"total"`
	Codes []task.CodeCount `json:"codes"`
	// mode is "summary" (--summary), "codes" (--code without a value),
	// "all" (--all) or "" (plain: bounded).
	mode string
	// code is the --code value, for the hints.
	code string
}

// lintReply fills findings, summary and next for lint (and edit): next is
// the fix for the first finding, or, when only the summary is shown, the
// command that lists the most frequent code.
func lintReply(reply Reply, fs []task.Finding, mode, code string) Reply {
	if fs == nil {
		fs = []task.Finding{}
	}
	sum := LintSummary{Total: len(fs), Codes: task.Summarize(fs, mode == "codes"), mode: mode, code: code}
	if sum.Codes == nil {
		sum.Codes = []task.CodeCount{}
	}
	reply["findings"] = fs
	reply["summary"] = sum
	if len(fs) == 0 {
		return reply
	}
	switch {
	case mode == "summary" || mode == "codes":
		reply["next"] = sum.Codes[0].List
	default:
		// With findings, next is the repair of the first one, never an
		// unrelated task (rx74bp).
		reply["next"] = fs[0].FixLine()
	}
	return reply
}

// lintFlags reads lint's flags: the mode and a --code value checked against
// the known codes. "--code help" lists the codes, like --code alone.
func lintFlags(f flags) (mode, code string, err error) {
	code = f.get("code")
	switch {
	case f.has("code") && (code == "" || code == "help"):
		return "codes", "", nil
	case f.has("summary"):
		mode = "summary"
	case f.has("all"):
		mode = "all"
	}
	if code != "" && !slices.Contains(task.CodeNames(), code) {
		e := task.Invalid("code", code, task.CodeNames())
		// The fix lists the codes with their counts: a guessed closest
		// code ("nope" -> "other") would run, but mean nothing.
		return "", "", usage("unknown code "+code+"; closest: "+e.Closest+"; valid: "+strings.Join(task.CodeNames(), ", "), "hug task lint --code")
	}
	return mode, code, nil
}

// writeFindings prints a lint or edit reply's findings: never nothing.
func writeFindings(w io.Writer, fs []task.Finding, sum LintSummary) {
	switch {
	case sum.mode == "codes":
		fmt.Fprintf(w, "finding codes (%s on this board):\n", plural(sum.Total, "finding", "findings"))
		writeCodes(w, sum.Codes)
		return
	case len(fs) == 0 && sum.code != "":
		fmt.Fprintf(w, "no findings with code %s\n", sum.code)
		return
	case len(fs) == 0:
		fmt.Fprintln(w, "no findings")
		return
	case sum.mode == "summary":
		fmt.Fprintf(w, "%s in %s:\n", plural(sum.Total, "finding", "findings"), plural(len(sum.Codes), "code", "codes"))
		writeCodes(w, sum.Codes)
		return
	}
	shown := fs
	if sum.mode != "all" && len(fs) > lintBound {
		if sum.code == "" {
			fmt.Fprintf(w, "%s in %s:\n", plural(sum.Total, "finding", "findings"), plural(len(sum.Codes), "code", "codes"))
			writeCodes(w, sum.Codes)
			fmt.Fprintf(w, "first %d:\n", lintShown)
		}
		shown = fs[:lintShown]
	}
	for _, v := range shown {
		fmt.Fprintln(w, v.String())
		fmt.Fprintln(w, "  fix: "+v.FixLine())
	}
	if len(shown) < len(fs) {
		if sum.code != "" {
			fmt.Fprintf(w, "%d of %d shown; every one: hug task lint --code %s --all\n", len(shown), len(fs), sum.code)
		} else {
			fmt.Fprintf(w, "%d of %d shown; every one: hug task lint --all; one kind: hug task lint --code CODE\n", len(shown), len(fs))
		}
	}
}

// writeCodes prints one line per code: count, code, meaning, the command
// that lists it, aligned with spaces (tabs show without width in jcode).
func writeCodes(w io.Writer, cs []task.CodeCount) {
	wn, wc, wm := 0, 0, 0
	for _, c := range cs {
		wn = max(wn, len(fmt.Sprint(c.Count)))
		wc = max(wc, len(c.Code))
		wm = max(wm, len(c.Meaning))
	}
	for _, c := range cs {
		fmt.Fprintf(w, "  %*d  %-*s  %-*s  %s\n", wn, c.Count, wc, c.Code, wm, c.Meaning, c.List)
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

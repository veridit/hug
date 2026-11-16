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
	// Hidden counts findings on closed tasks (done, dropped) left out
	// without --all; HiddenCodes counts them by code.
	Hidden      int              `json:"hidden"`
	HiddenCodes []task.CodeCount `json:"hidden_codes"`
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
	sum := LintSummary{Total: len(fs), Codes: task.Summarize(fs, mode == "codes"), HiddenCodes: []task.CodeCount{}, mode: mode, code: code}
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

// withCode keeps the findings of one code.
func withCode(fs []task.Finding, code string) []task.Finding {
	kept := []task.Finding{}
	for _, x := range fs {
		if task.FindingCode(x.Problem) == code {
			kept = append(kept, x)
		}
	}
	return kept
}

// setHidden records the findings left out because their tasks are closed.
// With nothing open to fix, next shows the hidden ones.
func setHidden(reply Reply, hidden []task.Finding, code string) {
	sum := reply["summary"].(LintSummary)
	sum.Hidden = len(hidden)
	if hc := task.Summarize(hidden, false); hc != nil {
		sum.HiddenCodes = hc
	}
	reply["summary"] = sum
	reply["hidden"] = sum.Hidden
	if sum.Total == 0 && sum.Hidden > 0 && sum.mode != "codes" {
		reply["next"] = hiddenCommand(code)
	}
}

func hiddenCommand(code string) string {
	if code != "" {
		return "hug task lint --code " + code + " --all"
	}
	return "hug task lint --all"
}

// writeHidden ends the output with what was left out, if anything:
// "986 findings on closed tasks hidden (432 done-unchecked, ...): hug task lint --all".
func writeHidden(w io.Writer, sum LintSummary) {
	if sum.Hidden == 0 {
		return
	}
	var parts []string
	for i, c := range sum.HiddenCodes {
		if i == 3 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%d %s", c.Count, c.Code))
	}
	fmt.Fprintf(w, "%s on closed tasks hidden (%s): %s\n", plural(sum.Hidden, "finding", "findings"), strings.Join(parts, ", "), hiddenCommand(sum.code))
}

// writeFindings prints a lint or edit reply's findings: never nothing.
func writeFindings(w io.Writer, fs []task.Finding, sum LintSummary) {
	defer writeHidden(w, sum)
	switch {
	case sum.mode == "codes":
		fmt.Fprintf(w, "finding codes (%s on this board):\n", plural(sum.Total, "finding", "findings"))
		writeCodes(w, sum.Codes)
		return
	case len(fs) == 0 && sum.code != "":
		fmt.Fprintf(w, "no findings with code %s%s\n", sum.code, onOpen(sum))
		return
	case len(fs) == 0:
		fmt.Fprintf(w, "no findings%s\n", onOpen(sum))
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

// onOpen qualifies "no findings" when closed tasks' findings were hidden.
func onOpen(sum LintSummary) string {
	if sum.Hidden > 0 {
		return " on open tasks"
	}
	return ""
}

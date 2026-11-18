package cli

import (
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// questionText is a numbered question in show (q2adr8): "question N to
// NAME (check C): TEXT", more lines indented, then its answer under it.
func questionText(q task.AskedQuestion) string {
	var b strings.Builder
	head := "question " + strconv.Itoa(q.N) + " to " + q.To
	var meta []string
	if q.Check > 0 {
		meta = append(meta, "check "+strconv.Itoa(q.Check))
	}
	if q.Show != "" {
		meta = append(meta, "show "+q.Show)
	}
	if len(meta) > 0 {
		head += " (" + strings.Join(meta, ", ") + ")"
	}
	lines := strings.Split(q.Text, "\n")
	b.WriteString(head + ": " + lines[0] + "\n")
	for _, l := range lines[1:] {
		b.WriteString("  " + l + "\n")
	}
	if q.Answer != nil {
		b.WriteString("  answered by " + q.Answer.By + ", " + strings.Replace(q.Answer.At, "T", " ", 1) + ":\n")
		for _, l := range strings.Split(q.Answer.Text, "\n") {
			b.WriteString("    " + l + "\n")
		}
	}
	// A withdrawn question (nanvnt) keeps its text for the record but
	// says so under it, so it never reads as one still waiting.
	if w := q.Withdrawn; w != nil && q.Answer == nil {
		line := "  withdrawn " + strings.Replace(w.At, "T", " ", 1)
		if w.Text != "" {
			line += " (" + w.Text + ")"
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

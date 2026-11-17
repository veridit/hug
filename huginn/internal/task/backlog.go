package task

import (
	"fmt"
	"sort"
	"strings"
)

// Backlog import leftovers (fkpyg3, Jørgen 2026-11-17): Backlog.md's
// <!-- SECTION:...:BEGIN/END --> and <!-- AC:... --> comment lines, and
// frontmatter fields hug never reads. Lint names them; hug upgrade removes
// them, the mechanical part only: it never rewrites a section's text.

// BacklogFields are the imported fields hug does not use: created_date is
// copied to created at import; ordinal became the file rank; backlog_status
// and import_type_inference record how status and type were mapped.
var BacklogFields = []string{"created_date", "ordinal", "backlog_status", "import_type_inference"}

func isBacklogMarker(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "<!-- SECTION:") || strings.HasPrefix(t, "<!-- AC:")
}

// backlogLeftovers are the frontmatter fields (in file order) and the
// marker lines (0-based indexes into d.lines) hug upgrade removes.
func (d *Document) backlogLeftovers() (fields []string, markers []int) {
	if d.frontEnd < 0 || !d.frontmatterValid {
		return nil, nil
	}
	for _, k := range BacklogFields {
		if _, ok := d.fields[k]; ok {
			fields = append(fields, k)
		}
	}
	sort.Slice(fields, func(i, j int) bool { return d.fields[fields[i]] < d.fields[fields[j]] })
	for n := d.frontEnd + 1; n < len(d.lines); n++ {
		if isBacklogMarker(d.lines[n]) {
			markers = append(markers, n)
		}
	}
	return fields, markers
}

// BacklogFindings names each leftover kind of one task, with hug upgrade as
// the fix.
func BacklogFindings(d *Document) []Finding {
	fields, markers := d.backlogLeftovers()
	var out []Finding
	if len(fields) > 0 {
		out = append(out, Finding{Path: d.Task.Path, Line: d.fields[fields[0]] + 1,
			Problem: "backlog leftover: unused fields " + strings.Join(fields, ", "), Fix: "hug upgrade"})
	}
	if len(markers) > 0 {
		out = append(out, Finding{Path: d.Task.Path, Line: markers[0] + 1,
			Problem: fmt.Sprintf("backlog leftover: %d <!-- SECTION:... --> marker lines", len(markers)), Fix: "hug upgrade"})
	}
	return out
}

// StripBacklog is the file without its backlog leftovers; changed is false
// when there were none (the bytes are then the same).
func (d *Document) StripBacklog() (data []byte, changed bool) {
	fields, markers := d.backlogLeftovers()
	if len(fields) == 0 && len(markers) == 0 {
		return d.Bytes(), false
	}
	drop := map[int]bool{}
	for _, k := range fields {
		n, end := d.fields[k], d.fieldEnds[k]
		if end <= n {
			end = n + 1
		}
		for i := n; i < end; i++ {
			drop[i] = true
		}
	}
	for _, n := range markers {
		drop[n] = true
	}
	var out []string
	for i, l := range d.lines {
		if !drop[i] {
			out = append(out, l)
		}
	}
	return []byte(strings.Join(out, "")), true
}

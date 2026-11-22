package cli

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veridit/hug/huginn/internal/task"
)

// BoardCard is one task on the printed board.
type BoardCard struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Assignee []string `json:"assignee,omitempty"`
	Model    string   `json:"model,omitempty"`
	Effort   string   `json:"effort,omitempty"`
	Blocked  bool     `json:"blocked,omitempty"`
}

// BoardColumn is one status column, in rank order. Hidden counts done tasks
// left out when the column is shortened.
type BoardColumn struct {
	State  string      `json:"state"`
	Cards  []BoardCard `json:"cards"`
	Total  int         `json:"total"`
	Hidden int         `json:"hidden,omitempty"`
}

const boardDoneTail = 5

// runBoard is the non-interactive board: todo, doing and done side by side,
// so a human or agent can see the state without the TUI.
func (a *App) runBoard(_ context.Context, s *session, f flags, filter task.Filter) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	ts, e := s.s.Tasks()
	if e != nil {
		return nil, e
	}
	cols := []BoardColumn{{State: "todo"}, {State: "doing"}, {State: "done"}}
	for _, t := range ts {
		// board takes no --state (its columns are the states); the other
		// filters apply, ! too (7zg9mf).
		if !filter.Keeps(t, ts) {
			continue
		}
		for i := range cols {
			if cols[i].State == t.State {
				cols[i].Cards = append(cols[i].Cards, BoardCard{ID: t.ID, Title: t.Title, Assignee: t.Assignee, Model: t.Model, Effort: t.Effort, Blocked: task.Waiting(t, ts)})
			}
		}
	}
	for i := range cols {
		cols[i].Total = len(cols[i].Cards)
		if cols[i].Cards == nil {
			cols[i].Cards = []BoardCard{}
		}
	}
	// Done grows without bound; the board shows the most recent end of it.
	if done := &cols[2]; !f.has("all") && len(done.Cards) > boardDoneTail {
		done.Hidden = len(done.Cards) - boardDoneTail
		done.Cards = done.Cards[len(done.Cards)-boardDoneTail:]
	}
	next := "hug task next"
	if len(cols[1].Cards) > 0 {
		next = "hug task show " + cols[1].Cards[0].ID
	}
	return Reply{"board": cols, "next": next}, nil
}

func boardWidth(columns string) int {
	if n, err := strconv.Atoi(columns); err == nil && n >= 60 {
		return n
	}
	return 120
}

func clip(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:max(0, n-1)]) + "…"
}

// writeBoard prints the columns side by side, one card per line.
func writeBoard(w io.Writer, cols []BoardColumn, total int) {
	width := max(18, (total-2*(len(cols)-1))/len(cols))
	cells := make([][]string, len(cols))
	rows := 0
	for i, c := range cols {
		head := fmt.Sprintf("%s (%d)", strings.ToUpper(c.State), c.Total)
		cells[i] = append(cells[i], clip(head, width), strings.Repeat("-", min(width, utf8.RuneCountInString(head))))
		for _, card := range c.Cards {
			mark := ""
			if card.Blocked {
				mark = "! "
			}
			cells[i] = append(cells[i], clip(card.ID+" "+mark+card.Title, width))
		}
		if c.Hidden > 0 {
			cells[i] = append(cells[i], clip(fmt.Sprintf("... %d more (--all)", c.Hidden), width))
		}
		rows = max(rows, len(cells[i]))
	}
	for r := 0; r < rows; r++ {
		var b strings.Builder
		for i := range cells {
			cell := ""
			if r < len(cells[i]) {
				cell = cells[i][r]
			}
			if i < len(cells)-1 {
				cell += strings.Repeat(" ", width-utf8.RuneCountInString(cell)+2)
			}
			b.WriteString(cell)
		}
		fmt.Fprintln(w, strings.TrimRight(b.String(), " "))
	}
	fmt.Fprintln(w, "! = blocked by an open dependency or a condition that does not hold")
}

// writeBoardWide prints each column as a section with full, unclipped
// titles (one card per line), for terminals too narrow for side-by-side
// columns or when titles matter.
func writeBoardWide(w io.Writer, cols []BoardColumn) {
	for i, c := range cols {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s (%d)\n", strings.ToUpper(c.State), c.Total)
		for _, card := range c.Cards {
			mark := " "
			if card.Blocked {
				mark = "!"
			}
			who := ""
			if len(card.Assignee) > 0 {
				who = "  [" + strings.Join(card.Assignee, ", ")
				label := card.Model
				if card.Effort != "" {
					label += "/" + card.Effort // jr55s8
				}
				if card.Model != "" && !slices.Contains(card.Assignee, card.Model) {
					who += " · " + label
				} else if card.Effort != "" && slices.Contains(card.Assignee, card.Model) {
					who = "  [" + strings.Join(card.Assignee, ", ") + "/" + card.Effort
				}
				who += "]"
			}
			fmt.Fprintf(w, "  %s %s %s%s\n", card.ID, mark, card.Title, who)
		}
		if c.Hidden > 0 {
			fmt.Fprintf(w, "  ... %d more (--all)\n", c.Hidden)
		}
	}
}

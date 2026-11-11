package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veridit/huginn/internal/task"
)

// BoardCard is one task on the printed board.
type BoardCard struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Assignee []string `json:"assignee,omitempty"`
	Blocked  bool     `json:"blocked,omitempty"`
}

// BoardColumn is one status column, in rank order. Hidden counts done tasks
// left out when the column is shortened.
type BoardColumn struct {
	Status string      `json:"status"`
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
	cols := []BoardColumn{{Status: "todo"}, {Status: "doing"}, {Status: "done"}}
	for _, t := range ts {
		if filter.Type != "" && t.Type != filter.Type || filter.Priority != "" && t.Priority != filter.Priority || filter.Group != "" && t.Group != filter.Group || filter.Label != "" && !slices.Contains(t.Labels, filter.Label) || filter.Mine != "" && !slices.Contains(t.Assignee, filter.Mine) {
			continue
		}
		for i := range cols {
			if cols[i].Status == t.Status {
				cols[i].Cards = append(cols[i].Cards, BoardCard{ID: t.ID, Title: t.Title, Assignee: t.Assignee, Blocked: len(task.Blocked(t, ts)) > 0})
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
	next := "huginn next"
	if len(cols[1].Cards) > 0 {
		next = "huginn show " + cols[1].Cards[0].ID
	}
	return Reply{"board": cols, "next": next}, nil
}

func boardWidth() int {
	if n, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && n >= 60 {
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
func writeBoard(w io.Writer, cols []BoardColumn) {
	width := max(18, (boardWidth()-2*(len(cols)-1))/len(cols))
	cells := make([][]string, len(cols))
	rows := 0
	for i, c := range cols {
		head := fmt.Sprintf("%s (%d)", strings.ToUpper(c.Status), c.Total)
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
	fmt.Fprintln(w, "! = blocked by an open dependency")
}

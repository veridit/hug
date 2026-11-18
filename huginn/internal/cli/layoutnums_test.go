package cli

import (
	"fmt"

	"github.com/veridit/hug/huginn/internal/task"
)

// The layout numbers the tests expect, from task.Layout, so the next
// layout bump changes no test that is not about it (w40cmk check 10).
var (
	// layoutLine is hug.toml's first line on a board at this hug's layout.
	layoutLine = fmt.Sprintf("layout = %d\n", task.Layout)
	// newerLayout is a board one layout ahead of this hug.
	newerLayout = task.Layout + 1
)

// upgradedFrom is hug upgrade's commit subject for a board with no task
// files to move (layout from -> this hug's).
func upgradedFrom(from int) string {
	return fmt.Sprintf("hug upgrade: board layout %d -> %d", from, task.Layout)
}

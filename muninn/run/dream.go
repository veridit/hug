package run

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/veridit/hug/muninn/internal/app"
)

// dreamCmd runs one dream pass: mechanical duplicates are retired with an
// undo receipt, judgement items are queued for agents (memory_dream).
func dreamCmd() *cobra.Command {
	var dry, list, asJSON, summary, whenQuiet, status, week bool
	c := &cobra.Command{Use: "dream", Short: "Tidy memories: retire exact duplicates (with undo), queue the rest for agents",
		Long: `dream runs one memory housekeeping pass.

Mechanical changes are applied automatically, with an undo receipt:
  near-identical duplicates in the same place, where the keeper keeps all text.
Everything that needs judgement is queued and handed to agents at a pause in
their work (session start, return after a break, wrap-up) through the
memory_dream MCP tool. Undo a pass with: hug mem tidy undo <receipt>.

dream also runs by itself: when a pass is due (one a day), the capture hook
starts a waiter. Once hug has seen a week of your turns, the waiter starts
the pass at the beginning of the day's quiet stretch, learned from an
hour-of-week histogram of when you type (hug mem dream --week). Until then
it runs the pass once you have been away 30 minutes. Either way the
machine's 5-minute load must be below 120. hug mem dream --status names the
last pass, its undo receipt, and the next quiet stretch.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if whenQuiet {
				return waitAndDream(cmd.Context())
			}
			return withApp(func(ctx context.Context, a *app.App) error {
				if status || week {
					lines := dreamStatus(a)
					if week {
						lines = dreamWeek(a)
					}
					for _, line := range lines {
						fmt.Fprintln(cmd.OutOrStdout(), line)
					}
					return nil
				}
				if summary {
					s := a.DreamSummarySince(time.Now().Add(-24 * time.Hour))
					if asJSON {
						return json.NewEncoder(cmd.OutOrStdout()).Encode(s)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "Last 24 h: %d duplicates merged, %d items handed to agents, %d resolved, %d open.\n", s.Retired, s.Handed, s.Resolved, s.Open)
					return nil
				}
				if list {
					cwd, _ := os.Getwd()
					items, err := a.Bound("", cwd).DreamItems(a, "cli", 5)
					if err != nil {
						return err
					}
					return json.NewEncoder(cmd.OutOrStdout()).Encode(items)
				}
				if !dry {
					release, err := app.DreamRun() // one pass at a time
					if err != nil {
						return err
					}
					defer release()
				}
				r, err := a.Dream(ctx, dry)
				if err != nil {
					return err
				}
				if !dry {
					if _, err := app.RecordDream(a.DB, app.DreamClock(), "by hand", r); err != nil {
						return err
					}
				}
				if asJSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(r)
				}
				verb := "Retired"
				if dry {
					verb = "Would retire"
					for _, line := range dreamStatus(a) {
						fmt.Fprintln(cmd.OutOrStdout(), line)
					}
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %d duplicate memories. Queued %d new items for agents; %d waiting.\n", verb, r.Retired, r.Queued, r.Open)
				if r.Backup != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Undo: hug mem tidy undo %s\n", shellQuote(r.Backup))
				}
				return nil
			})
		}}
	c.Flags().BoolVar(&dry, "dry-run", false, "report what would change, write nothing")
	c.Flags().BoolVar(&list, "list", false, "show the items an agent here would be handed (marks them handed)")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	c.Flags().BoolVar(&status, "status", false, "when dream last ran, what it did, its undo receipt, and when it runs next")
	c.Flags().BoolVar(&week, "week", false, "your week: turns per hour of the week, and the quiet stretches dream starts in")
	c.Flags().BoolVar(&whenQuiet, "when-quiet", false, "wait until you are away and the machine is idle, then run one pass (the capture hook starts this)")
	_ = c.Flags().MarkHidden("when-quiet")
	c.Flags().BoolVar(&summary, "summary", false, "one line: merged, handed out, resolved and open in the last 24 hours")
	return c
}

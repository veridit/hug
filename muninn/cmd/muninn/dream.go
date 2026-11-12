package main

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
	var dry, list, asJSON, summary bool
	c := &cobra.Command{Use: "dream", Short: "Tidy memories: retire exact duplicates (with undo), queue the rest for agents",
		Long: `dream runs one memory housekeeping pass.

Mechanical changes are applied automatically, with an undo receipt:
  near-identical duplicates in the same place, where the keeper keeps all text.
Everything that needs judgement is queued and handed to agents at a pause in
their work (session start, return after a break, wrap-up) through the
memory_dream MCP tool. Undo a pass with: muninn tidy undo <receipt>.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
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
				r, err := a.Dream(ctx, dry)
				if err != nil {
					return err
				}
				if asJSON {
					return json.NewEncoder(cmd.OutOrStdout()).Encode(r)
				}
				verb := "Retired"
				if dry {
					verb = "Would retire"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s %d duplicate memories. Queued %d new items for agents; %d waiting.\n", verb, r.Retired, r.Queued, r.Open)
				if r.Backup != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Undo: muninn tidy undo %s\n", shellQuote(r.Backup))
				}
				return nil
			})
		}}
	c.Flags().BoolVar(&dry, "dry-run", false, "report what would change, write nothing")
	c.Flags().BoolVar(&list, "list", false, "show the items an agent here would be handed (marks them handed)")
	c.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	c.Flags().BoolVar(&summary, "summary", false, "one line: merged, handed out, resolved and open in the last 24 hours")
	return c
}

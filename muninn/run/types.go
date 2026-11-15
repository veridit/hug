package run

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/veridit/hug/muninn/internal/app"
)

func typesCmd() *cobra.Command {
	var limit, batch int
	var dry, asJSON bool
	c := &cobra.Command{Use: "types", Short: "Infer the type (rule, preference, fact, record) of memories that have none, with the capture model",
		Long: "Asks the capture model (extract config) for the type of every memory without one, 12 per call, and records it by\n" +
			"content hash. A type stated on the memory always wins. Resumable: typed memories are skipped. --dry-run counts only.",
		RunE: func(*cobra.Command, []string) error {
			return withApp(func(ctx context.Context, a *app.App) error {
				r, err := a.InferTypes(ctx, limit, batch, dry, func(done, total int) {
					if done%240 == 0 || done == total {
						fmt.Fprintf(os.Stderr, "JCODE_PROGRESS {\"current\":%d,\"total\":%d,\"unit\":\"memories\"}\n", done, total)
					}
				})
				if asJSON {
					b, _ := json.MarshalIndent(r, "", "  ")
					fmt.Println(string(b))
				} else if dry {
					fmt.Printf("%d memories have no type; about %d model calls\n", r.Untyped, (r.Untyped+max(batch, 1)-1)/max(batch, 1))
				} else {
					fmt.Printf("typed %d (rule %d, preference %d, fact %d, record %d), failed %d, %d calls, %d+%d tokens; %d still untyped\n",
						r.Typed, r.Counts["rule"], r.Counts["preference"], r.Counts["fact"], r.Counts["record"], r.Failed, r.Calls,
						r.Usage.PromptTokens, r.Usage.CompletionTokens, r.Remaining)
				}
				return err
			})
		}}
	c.Flags().IntVar(&limit, "limit", 0, "at most this many memories (0 = all)")
	c.Flags().IntVar(&batch, "batch", 12, "memories per model call")
	c.Flags().BoolVar(&dry, "dry-run", false, "count untyped memories, call nothing")
	c.Flags().BoolVar(&asJSON, "json", false, "JSON report")
	return c
}

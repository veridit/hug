package run

import (
	"time"

	"github.com/veridit/hug/huginn/internal/cli"
)

// SyncTimes are the background sync's two waits (0hkaa5): a burst of
// writes is pushed once it is quiet for settle, and a read fetches when the
// last pass is older than fetchAfter. Another clone sees a change at its
// next command, at most about settle + fetchAfter later.
func SyncTimes() (settle, fetchAfter time.Duration) { return cli.Settle, cli.FetchAfter }

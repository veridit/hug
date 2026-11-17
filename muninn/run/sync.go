package run

import (
	"time"

	"github.com/veridit/hug/muninn/internal/project"
)

// SyncTimes are the project-memory background sync's two waits (0hkaa5),
// as huginn's: settle before a push, fetchAfter between a read's fetches.
func SyncTimes() (settle, fetchAfter time.Duration) { return project.Settle, project.FetchAfter }

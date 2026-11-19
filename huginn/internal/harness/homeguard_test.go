package harness

import (
	"os"
	"testing"

	"github.com/veridit/hug/internal/homeguard"
)

// No test reaches this machine's real harness configs (ynh84t).
func TestMain(m *testing.M) { os.Exit(homeguard.Run(m)) }

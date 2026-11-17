//go:build linux

package cond

import (
	"os"
	"strconv"
	"strings"
)

// systemLoad reads the 5-minute average from /proc/loadavg.
func systemLoad() (float64, bool) {
	b, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, false
	}
	f := strings.Fields(string(b))
	if len(f) < 2 {
		return 0, false
	}
	v, err := strconv.ParseFloat(f[1], 64)
	return v, err == nil
}

//go:build darwin

package cond

import (
	"encoding/binary"

	"golang.org/x/sys/unix"
)

// systemLoad reads vm.loadavg: three uint32 averages (1, 5, 15 min), padding,
// then the fixed-point scale as a long.
func systemLoad() (float64, bool) {
	b, err := unix.SysctlRaw("vm.loadavg")
	if err != nil || len(b) < 24 {
		return 0, false
	}
	scale := float64(binary.LittleEndian.Uint64(b[16:24]))
	if scale == 0 {
		return 0, false
	}
	return float64(binary.LittleEndian.Uint32(b[4:8])) / scale, true
}

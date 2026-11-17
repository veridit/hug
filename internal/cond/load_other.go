//go:build !darwin && !linux

package cond

// systemLoad: Windows and others have no load average; a load condition
// there is unknown and keeps waiting (lint says so).
func systemLoad() (float64, bool) { return 0, false }

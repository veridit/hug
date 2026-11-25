//go:build windows

package main

// fileInode: hug version --running lists no processes on Windows.
func fileInode(string) uint64 { return 0 }

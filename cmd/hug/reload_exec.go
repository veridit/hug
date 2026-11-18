//go:build !windows

package main

import "syscall"

// execHug replaces this process with path, keeping pid and fds 0-2.
var execHug = func(path string, argv, env []string) error { return syscall.Exec(path, argv, env) }

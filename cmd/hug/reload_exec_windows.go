//go:build windows

package main

// execHug is nil on Windows: there is no exec, so a changed hug stops the
// server between requests and the app starts the new one (mbdfzs).
var execHug func(path string, argv, env []string) error

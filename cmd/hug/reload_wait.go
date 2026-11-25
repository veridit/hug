//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package main

import (
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// pollStdin is hug mcp's stdin as an idleSource (rvb3j5): poll(2) on fd 0
// says when the client wrote, without taking its bytes, so an idle reload
// leaves them in the pipe for the new hug. SIGHUP wakes the wait through a
// pipe of our own (close-on-exec: the new hug never sees it).
type pollStdin struct {
	f    *os.File
	fd   int
	wake *os.File
	wfd  int
}

// idleStdin makes f waitable, and SIGHUP a nudge to look at the hug now.
// When it cannot, f is returned as is: the reload then happens at a request.
func idleStdin(f *os.File) io.Reader {
	r, w, err := os.Pipe()
	if err != nil {
		return f
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	go func() {
		for range sig {
			w.Write([]byte{1})
		}
	}()
	return &pollStdin{f: f, fd: int(f.Fd()), wake: r, wfd: int(r.Fd())}
}

func (p *pollStdin) Read(b []byte) (int, error) { return p.f.Read(b) }

func (p *pollStdin) wait(d time.Duration) (ready, nudged bool, err error) {
	ms := int(d / time.Millisecond)
	if ms < 1 {
		ms = 1
	}
	fds := []unix.PollFd{{Fd: int32(p.fd), Events: unix.POLLIN}, {Fd: int32(p.wfd), Events: unix.POLLIN}}
	if _, err := unix.Poll(fds, ms); err != nil {
		if err == unix.EINTR {
			return false, false, nil
		}
		return false, false, err
	}
	if fds[1].Revents != 0 {
		var b [64]byte
		p.wake.Read(b[:])
		nudged = true
	}
	return fds[0].Revents != 0, nudged, nil
}

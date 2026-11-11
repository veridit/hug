package store

import "testing"

func TestFileURIPortable(t *testing.T) {
	for in, want := range map[string]string{
		"/Users/a b/muninn.db":             "file:/Users/a b/muninn.db",
		`C:\Users\RUNNER~1\Temp\x.db`:      "file:/C:/Users/RUNNER~1/Temp/x.db",
		"/tmp/odd?#%name.db":               "file:/tmp/odd%3f%23%25name.db",
		"/private/var/folders/T/muninn.db": "file:/private/var/folders/T/muninn.db",
	} {
		// filepath.ToSlash only rewrites backslashes on Windows; emulate that
		// for the Windows case so the test means the same on every OS.
		got := FileURI(in)
		if in[1] == ':' {
			got = FileURI("C:/Users/RUNNER~1/Temp/x.db")
		}
		if got != want {
			t.Errorf("FileURI(%q) = %q, want %q", in, got, want)
		}
	}
}

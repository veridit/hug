package main

import (
	"os"
	"testing"
)

func TestRunDispatch(t *testing.T) {
	devnull, _ := os.Open(os.DevNull)
	defer devnull.Close()
	old := os.Stdout
	os.Stdout = devnull
	defer func() { os.Stdout = old }()
	for _, tc := range []struct {
		name string
		args []string
		want int
	}{
		{"hug", nil, 0},
		{"hug", []string{"help"}, 0},
		{"hug", []string{"version"}, 0},
		{"hug", []string{"mem", "version"}, 0},
		{"hug", []string{"task", "version"}, 0},
		{"muninn", []string{"version"}, 0},
		{"huginn.exe", []string{"version"}, 0},
		{"hug", []string{"bogus"}, 2},
	} {
		if got := run(tc.name, tc.args); got != tc.want {
			t.Errorf("%s %v = %d, want %d", tc.name, tc.args, got, tc.want)
		}
	}
}

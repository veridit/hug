package cli

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// m8shdj check 1: 20 parallel hug task note commands on one board all
// succeed and all land, also on a board without a remote: writes take the
// board turn instead of colliding on git's index.lock.
func TestParallelNotesWithoutRemoteAllLand(t *testing.T) {
	a := app(t)
	execute(t, a, "init")
	id := execute(t, a, "new", "Parallel", "--goal", "G", "--accept", "C")["task"].(Reply)["id"].(string)
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := &App{Cwd: a.Cwd, In: strings.NewReader(""), Out: &bytes.Buffer{}, Err: &bytes.Buffer{}}
			if _, e := c.Execute(context.Background(), []string{"note", id, "parallel " + strconv.Itoa(i)}); e != nil {
				errs <- e
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Errorf("parallel note: %v", e)
	}
	if n := strings.Count(taskFile(t, a, id), "parallel "); n != 20 {
		t.Fatalf("%d of 20 notes in the task file", n)
	}
	if out := git(t, a.Cwd+"/"+TaskDir, "status", "--porcelain"); out != "" {
		t.Fatalf("board left uncommitted: %s", out)
	}
}

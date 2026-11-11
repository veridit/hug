package ingest

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func TestConcurrentRememberKeepsEveryWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memories.json")
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				if _, _, err := Remember(path, fmt.Sprintf("writer %d note %d", w, i), "fact", "global", nil); err != nil {
					t.Error(err)
				}
			}
		}(w)
	}
	wg.Wait()
	mems, err := LoadNative(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(mems) != 40 {
		t.Fatalf("stored %d of 40", len(mems))
	}
}

func TestMergeSaveKeepsOtherWriters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memories.json")
	a, _, _ := Remember(path, "kept and edited", "fact", "global", nil)
	b, _, _ := Remember(path, "removed by batch", "fact", "global", nil)
	c, _, _ := Remember(path, "edited by other", "fact", "global", nil)
	base, _ := LoadNative(path)
	edited := CloneNative(base)
	// The long batch edits a, drops b and adds d.
	var keep []NativeMemory
	for _, m := range edited {
		switch m.ID {
		case a.ID:
			m.Strength = 9
		case b.ID:
			continue
		}
		keep = append(keep, m)
	}
	keep = append(keep, NativeMemory{ID: "d", Content: "added by batch", Active: true})
	// Meanwhile another process adds e and changes c.
	Remember(path, "added by other", "fact", "global", nil)
	if err := UpdateNative(path, func(cur []NativeMemory) ([]NativeMemory, bool, error) {
		for i := range cur {
			if cur[i].ID == c.ID {
				cur[i].Active = false
			}
		}
		return cur, true, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := MergeSave(path, base, keep); err != nil {
		t.Fatal(err)
	}
	got := map[string]NativeMemory{}
	mems, _ := LoadNative(path)
	for _, m := range mems {
		got[m.Content] = m
	}
	if got["kept and edited"].Strength != 9 {
		t.Error("batch edit lost")
	}
	if _, ok := got["removed by batch"]; ok {
		t.Error("batch removal lost")
	}
	if got["edited by other"].Active {
		t.Error("other writer's change overwritten")
	}
	if _, ok := got["added by other"]; !ok {
		t.Error("other writer's memory lost")
	}
	if _, ok := got["added by batch"]; !ok {
		t.Error("batch addition lost")
	}
}

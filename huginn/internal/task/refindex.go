package task

import (
	"strings"
	"sync"
)

// refIndex answers ResolveDependency without scanning the board: lowercase
// id and alias to the positions of the tasks that carry them. At 10,000
// tasks a scan per dependency made lint, next and ls quadratic (44w1ev:
// ResolveDependency was 16% of lint and most of next).
type refIndex struct {
	ids, aliases map[string][]int
}

func buildRefIndex(tasks []Task) *refIndex {
	x := &refIndex{ids: make(map[string][]int, len(tasks)), aliases: map[string][]int{}}
	for i, t := range tasks {
		k := strings.ToLower(t.ID)
		x.ids[k] = append(x.ids[k], i)
		for _, a := range t.Aliases {
			k := strings.ToLower(a)
			// One task naming the same alias twice still counts once.
			if l := x.aliases[k]; len(l) == 0 || l[len(l)-1] != i {
				x.aliases[k] = append(x.aliases[k], i)
			}
		}
	}
	return x
}

// The last board indexed, keyed by its first element and length, as
// ChildIndex's callers do: commands pass the same slice many times.
var refMemo struct {
	sync.Mutex
	first *Task
	n     int
	idx   *refIndex
}

func refIndexFor(tasks []Task, rebuild bool) *refIndex {
	if len(tasks) < 64 {
		return nil // a scan is cheaper than a map for small boards
	}
	refMemo.Lock()
	defer refMemo.Unlock()
	if rebuild || refMemo.first != &tasks[0] || refMemo.n != len(tasks) {
		refMemo.first, refMemo.n, refMemo.idx = &tasks[0], len(tasks), buildRefIndex(tasks)
	}
	return refMemo.idx
}

// What the index says about one ref.
const (
	refHit   = iota // answered
	refStale        // a position no longer names ref: rebuild the index
	refMiss         // not answered here (no owner, or several aliases): scan
)

// resolve is ResolveDependency over the index, with the same answers.
func (x *refIndex) resolve(tasks []Task, ref string) (Task, error, int) {
	k := strings.ToLower(ref)
	for _, i := range x.ids[k] {
		if i >= len(tasks) || !strings.EqualFold(tasks[i].ID, ref) {
			return Task{}, nil, refStale
		}
	}
	for _, i := range x.aliases[k] {
		if i >= len(tasks) || !containsFold(tasks[i].Aliases, ref) {
			return Task{}, nil, refStale
		}
	}
	ids := x.ids[k]
	if len(ids) == 1 {
		return tasks[ids[0]], nil, refHit
	}
	if len(ids) > 1 {
		return Task{}, &Error{Kind: "duplicate-id", ID: ref, Fix: "hug task lint"}, refHit
	}
	// An alias that is also some task's id was answered above; here the
	// alias owners alone decide, as the scan does.
	if found := x.aliases[k]; len(found) == 1 {
		return tasks[found[0]], nil, refHit
	}
	return Task{}, nil, refMiss
}

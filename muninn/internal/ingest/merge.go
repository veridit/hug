package ingest

import "reflect"

// MergeSave writes edits made to base, a copy loaded earlier, onto the store
// as it is now. Memories another process added or changed since base was
// loaded are kept: only entries that differ between base and edited are
// written (changed, added, or removed). Use it for long operations such as
// capture or tidy, which must not hold the write lock across model calls.
func MergeSave(path string, base, edited []NativeMemory) error {
	return UpdateNative(path, func(cur []NativeMemory) ([]NativeMemory, bool, error) {
		return mergeEdits(cur, base, edited)
	})
}

func mergeEdits(cur, base, edited []NativeMemory) ([]NativeMemory, bool, error) {
	before := make(map[string]NativeMemory, len(base))
	for _, m := range base {
		before[m.ID] = m
	}
	after := make(map[string]bool, len(edited))
	changed := map[string]NativeMemory{}
	for _, m := range edited {
		after[m.ID] = true
		if old, ok := before[m.ID]; !ok || !reflect.DeepEqual(old, m) {
			changed[m.ID] = m
		}
	}
	removed := map[string]bool{}
	for id := range before {
		if !after[id] {
			removed[id] = true
		}
	}
	if len(changed) == 0 && len(removed) == 0 {
		return cur, false, nil
	}
	out := make([]NativeMemory, 0, len(cur)+len(changed))
	for _, m := range cur {
		if removed[m.ID] {
			continue
		}
		if n, ok := changed[m.ID]; ok {
			m = n
			delete(changed, m.ID)
		}
		out = append(out, m)
	}
	for _, m := range edited {
		if n, ok := changed[m.ID]; ok {
			out = append(out, n)
		}
	}
	return out, true, nil
}

// cloneNative copies mems deeply enough that edits to the copy (fields and
// slices) do not reach the original.
func CloneNative(mems []NativeMemory) []NativeMemory {
	out := make([]NativeMemory, len(mems))
	for i, m := range mems {
		m.Tags = append([]string(nil), m.Tags...)
		m.CaptureEvents = append([]string(nil), m.CaptureEvents...)
		out[i] = m
	}
	return out
}

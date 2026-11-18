package task

import (
	"reflect"
	"testing"
)

func TestLayout3NamesKeepsTaskNamedAttachmentInItsFolder(t *testing.T) {
	paths := []string{"later/a.8q6v7b.parent.md", "later/8q6v7b/b.7abcde.child.md", "later/8q6v7b/plain.txt"}
	want := map[string]string{
		paths[0]: "8/8q6v7b.md",
		paths[1]: "8/8q6v7b/b.7abcde.child.md",
		paths[2]: "8/8q6v7b/plain.txt",
	}
	if got := Layout3Names(paths); !reflect.DeepEqual(got, want) {
		t.Fatalf("attachment translated as another task: got %v, want %v", got, want)
	}
}

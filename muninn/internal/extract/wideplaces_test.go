package extract

import (
	"strings"
	"testing"
)

// promptPlacesSHA is sha256 of the capture system prompt with the global
// session's places (here, global) on origin/main 8bf24c02, before rnpdnm.
// With wide places off the prompt must stay byte-identical (crab, rnpdnm:
// the same pattern as extract.triggers).
const promptPlacesSHA = "7a83d75241b8e8d5edcf801e352843c76290062b85a0fc2d0198e3335a93450b"

var sessionPlaces = []Place{{"here", "this project or folder; code, repo, paths, tickets and deployment"}, {"global", "the user, preferences and tools across all their work"}}

func TestCapturePromptWidePlacesOffIsByteIdentical(t *testing.T) {
	if Defaults().WidePlaces {
		t.Fatal("wide places must be off by default")
	}
	if got := sha(SystemPrompt(Defaults(), nil)); got != promptSHA {
		t.Fatalf("off prompt changed: sha256 %s, want %s", got, promptSHA)
	}
	if got := sha(SystemPrompt(Defaults(), sessionPlaces)); got != promptPlacesSHA {
		t.Fatalf("off prompt with places changed: sha256 %s, want %s", got, promptPlacesSHA)
	}
	// On, but the session offered no personal place (a client session):
	// the prompt is unchanged too.
	on := Defaults()
	on.WidePlaces = true
	if got := sha(SystemPrompt(on, sessionPlaces)); got != promptPlacesSHA {
		t.Fatalf("on without a personal place changed the prompt: %s", got)
	}
	// On with the wide places: the list names them, the guidance follows,
	// and no scope string reaches the model.
	p := SystemPrompt(on, append(append([]Place{}, sessionPlaces...), WidePlaces...))
	for _, w := range []string{"\npersonal: ", "\nother: ", "Choose personal for", "place_name", "is global"} {
		if !strings.Contains(p, w) {
			t.Errorf("on prompt lacks %q", w)
		}
	}
	for _, w := range []string{"project:rc-", "client:", "\"personal\"", "jcode@", "folder:/"} {
		if strings.Contains(p[len(prompt):], w) {
			t.Errorf("on prompt's placement part names %q", w)
		}
	}
}

// place_name is read, cleaned to one short line, and dropped (never the
// memory) when malformed.
func TestParsePlaceName(t *testing.T) {
	base := `{"category":"fact","trust":"high","content":"Synthetic statbus deploys run from the release branch only","place":"other","place_name":%s}`
	for value, want := range map[string]string{
		`"Statbus"`:                         "Statbus",
		`"  Seid   lang "`:                  "Seid lang",
		`"project:rc-1234"`:                 "",
		`"` + strings.Repeat("x", 61) + `"`: "",
		`42`:                                "",
	} {
		cs, rejected := Parse(strings.Replace(base, "%s", value, 1))
		if value == `42` {
			// A wrong type is a schema error like any other field.
			if len(cs) != 0 || rejected != 1 {
				t.Errorf("place_name 42: %+v %d", cs, rejected)
			}
			continue
		}
		if rejected != 0 || len(cs) != 1 || cs[0].Place != "other" || cs[0].PlaceName != want {
			t.Errorf("place_name %s: %+v rejected=%d, want %q", value, cs, rejected, want)
		}
	}
}

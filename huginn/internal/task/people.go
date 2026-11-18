package task

import (
	"os"
	"sort"
	"strings"
)

// Person is one entry of hug.toml [people] (89y0b7, Jørgen 15:31): the
// short name rows show, the long name the detail shows, and every email
// they commit with. A person is found by git email; an agent is always
// its session name, since in Seid agents commit with Jørgen's email.
//
//	[people.jhf]
//	name = "Jørgen H. Fjeld"
//	email = ["jorgen@veridit.no", "12345+jhf@users.noreply.github.com"]
type Person struct {
	Short string   `json:"short"`
	Name  string   `json:"name,omitempty"`
	Email []string `json:"email,omitempty"`
	// Alias are other names the person is called by on this board
	// (agxqax): hug.toml alias = ["jorgen"]. Asking or waiting on one
	// resolves to Short; rows and asked_for always say Short.
	Alias []string `json:"alias,omitempty"`
}

// People reads hug.toml [people.SHORT] tables, in the order written. A
// missing file or section is no people. The one-line form
// [people] jhf = "a@b" is read too, as a person with one email.
func People(root string) ([]Person, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	var out []Person
	cur := -1
	flat := false
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			head := strings.TrimSpace(strings.Trim(line, "[]"))
			cur, flat = -1, head == "people"
			if short, ok := strings.CutPrefix(head, "people."); ok {
				out = append(out, Person{Short: scalar(short)})
				cur = len(out) - 1
			}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		switch {
		case flat:
			out = append(out, Person{Short: scalar(key), Email: list(value)})
		case cur >= 0 && key == "name":
			out[cur].Name = scalar(value)
		case cur >= 0 && key == "email":
			out[cur].Email = list(value)
		case cur >= 0 && key == "alias":
			out[cur].Alias = list(value)
		}
	}
	return out, nil
}

// tomlStrings reads a TOML string or array of strings.
func tomlStrings(value string) []string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "[") {
		if v := scalar(value); v != "" {
			return []string{v}
		}
		return nil
	}
	value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
	var out []string
	for _, v := range strings.Split(value, ",") {
		if v = scalar(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// PersonByEmail is the person who commits with email, matched without
// case; ok is false when nobody lists it.
func PersonByEmail(people []Person, email string) (Person, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return Person{}, false
	}
	for _, p := range people {
		for _, e := range p.Email {
			if strings.ToLower(strings.TrimSpace(e)) == email {
				return p, true
			}
		}
	}
	return Person{}, false
}

// PersonByName is the person whose short name is name, without case.
func PersonByName(people []Person, name string) (Person, bool) {
	name = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(name), "@"))
	for _, p := range people {
		if strings.EqualFold(p.Short, name) {
			return p, true
		}
	}
	return Person{}, false
}

// EmailShort is what a person with no [people] entry is called: the part
// of the email before the @.
func EmailShort(email string) string {
	local, _, _ := strings.Cut(strings.TrimSpace(email), "@")
	// GitHub's private form is 12345+name@users.noreply.github.com.
	if _, after, ok := strings.Cut(local, "+"); ok && after != "" {
		local = after
	}
	return local
}

// PeopleEntry is the hug.toml text for one person, as setup and lint
// offer it.
func PeopleEntry(p Person) string {
	q := func(s string) string { return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"` }
	emails := make([]string, len(p.Email))
	for i, e := range p.Email {
		emails[i] = q(e)
	}
	sort.Strings(emails)
	out := "[people." + p.Short + "]\n"
	if p.Name != "" {
		out += "name = " + q(p.Name) + "\n"
	}
	if len(p.Alias) > 0 {
		aliases := make([]string, len(p.Alias))
		for i, a := range p.Alias {
			aliases[i] = q(a)
		}
		out += "alias = [" + strings.Join(aliases, ", ") + "]\n"
	}
	return out + "email = [" + strings.Join(emails, ", ") + "]\n"
}

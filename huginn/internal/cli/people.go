package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// hug task people (89y0b7, Jørgen 15:31): who the people on this board
// are, in hug.toml [people.SHORT] with a long name and every email they
// commit with. A person is found by git email and named by the short name;
// an agent is always its own name. Without arguments it lists them and
// offers an entry for each commit author no entry covers, as a command;
// "add SHORT EMAIL [--name NAME]" writes one (adding an email to an
// existing entry) and commits it on the task branch.

// runPeople is hug task people [add SHORT EMAIL [--name NAME]].
func (a *App) runPeople(ctx context.Context, s *session, f flags) (Reply, error) {
	root := s.r.Dir
	people, e := task.People(root)
	if e != nil {
		return nil, e
	}
	if len(f.args) == 0 {
		offers := peopleOffers(ctx, a.Cwd, people)
		var fixes []string
		for _, p := range offers {
			fixes = append(fixes, addCommand(p))
		}
		next := "hug task next"
		if len(fixes) > 0 {
			next = fixes[0]
		}
		return Reply{"people": people, "offers": offers, "fixes": fixes, "next": next}, nil
	}
	if f.args[0] != "add" || len(f.args) != 3 {
		return nil, f.wrongCount("people takes no arguments, or add SHORT EMAIL")
	}
	short, email := strings.TrimSpace(f.args[1]), strings.TrimSpace(f.args[2])
	if short == "" || strings.ContainsAny(short, " \t.[]=\"") {
		return nil, usage("a short name is one word, as rows show it: "+short, "hug task people add "+task.EmailShort(email)+" "+email)
	}
	if !strings.Contains(email, "@") {
		return nil, usage("not an email: "+email, "hug task people add "+short+" "+short+"@example.org")
	}
	if p, ok := task.PersonByEmail(people, email); ok && !strings.EqualFold(p.Short, short) {
		return nil, usage(email+" is already "+p.Short+"'s", "hug task people")
	}
	p, found := task.PersonByName(people, short)
	if !found {
		p = task.Person{Short: short}
	}
	if n := strings.TrimSpace(f.get("name")); n != "" {
		p.Name = n
	}
	has := false
	for _, x := range p.Email {
		has = has || strings.EqualFold(x, email)
	}
	if !has {
		p.Email = append(p.Email, email)
	}
	if e = writePerson(root, p); e != nil {
		return nil, e
	}
	if e = s.commit("hug task people add " + short); e != nil {
		return nil, e
	}
	people, _ = task.People(root)
	return Reply{"people": people, "changed": true, "next": "hug task people"}, nil
}

// addCommand is the command that writes p.
func addCommand(p task.Person) string {
	cmd := "hug task people add " + p.Short + " " + p.Email[0]
	if p.Name != "" {
		cmd += ` --name "` + strings.ReplaceAll(p.Name, `"`, `\"`) + `"`
	}
	for _, e := range p.Email[1:] {
		cmd += " && hug task people add " + p.Short + " " + e
	}
	return cmd
}

// peopleOffers are entries for commit authors of the code repository that
// no [people] entry covers: emails grouped by author name, the short name
// from the first email (Jørgen 15:33).
func peopleOffers(ctx context.Context, cwd string, people []task.Person) []task.Person {
	out, e := exec.CommandContext(ctx, "git", "-C", cwd, "log", "--format=%aN%x09%aE", "-n", "2000").Output()
	if e != nil {
		return nil
	}
	byName := map[string]*task.Person{}
	var order []string
	for _, line := range strings.Split(string(out), "\n") {
		name, email, ok := strings.Cut(line, "\t")
		email = strings.TrimSpace(email)
		if !ok || email == "" {
			continue
		}
		if _, listed := task.PersonByEmail(people, email); listed {
			continue
		}
		p := byName[name]
		if p == nil {
			p = &task.Person{Short: shortFor(email), Name: strings.TrimSpace(name)}
			byName[name] = p
			order = append(order, name)
		}
		dup := false
		for _, x := range p.Email {
			dup = dup || strings.EqualFold(x, email)
		}
		if !dup {
			p.Email = append(p.Email, email)
		}
	}
	var offers []task.Person
	for _, n := range order {
		p := *byName[n]
		sort.Strings(p.Email)
		// The short name from the plainest email: a real address before a
		// noreply one (12345+jhf@users.noreply.github.com gives jhf anyway).
		p.Short = shortFor(p.Email[0])
		for _, e := range p.Email {
			if !strings.Contains(e, "noreply") {
				p.Short = shortFor(e)
				break
			}
		}
		if listed, ok := task.PersonByName(people, p.Short); ok {
			p.Short = listed.Short // more emails for someone listed
		}
		offers = append(offers, p)
	}
	return offers
}

// shortFor is the short name an email suggests: the part before the @,
// lower case, letters and digits only.
func shortFor(email string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(task.EmailShort(email)) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "person"
	}
	return b.String()
}

// writePerson puts p's table in hug.toml, replacing its old table, and
// keeps every other line as it was.
func writePerson(root string, p task.Person) error {
	path := filepath.Join(root, task.ConfigName)
	b, e := os.ReadFile(path)
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	if len(b) == 0 {
		lines = nil
	}
	var keep []string
	skip := false
	head := "[people." + p.Short + "]"
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			skip = strings.EqualFold(t, head)
		}
		if !skip {
			keep = append(keep, l)
		}
	}
	text := strings.TrimRight(strings.Join(keep, "\n"), "\n")
	if text != "" {
		text += "\n\n"
	}
	text += task.PeopleEntry(p)
	return os.WriteFile(path, []byte(text), 0o644)
}

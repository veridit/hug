package task

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"strings"
)

func (d *Document) parseFrontmatter() {
	var root yaml.Node
	source := strings.Join(d.lines[1:d.frontEnd], "")
	if e := yaml.Unmarshal([]byte(source), &root); e != nil {
		d.add(2, "malformed frontmatter: "+e.Error())
		return
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		d.add(2, "frontmatter must be a mapping")
		return
	}
	mapping := root.Content[0]
	d.frontmatterValid = true
	for n := 0; n+1 < len(mapping.Content); n += 2 {
		key, value := mapping.Content[n], mapping.Content[n+1]
		k := key.Value
		if key.Column != 1 {
			d.frontmatterValid = false
			d.add(key.Line+1, "frontmatter fields must start on separate lines")
		}
		if key.Kind != yaml.ScalarNode {
			d.add(key.Line+1, "frontmatter keys must be strings")
			continue
		}
		if _, ok := d.fields[k]; ok {
			d.add(key.Line+1, "duplicate field "+k)
		}
		d.fields[k] = key.Line
		end := d.frontEnd
		if n+2 < len(mapping.Content) {
			end = mapping.Content[n+2].Line
		}
		for end > key.Line+1 {
			// Blank lines and comments between this value and the next key
			// are not the value's. A comment is at the key's column: an
			// indented line, "#" or not, is a block scalar's own text
			// (t3fqaw: Seid's t6j13s title went on "  #return/#break
			// clauses ...", which set --title then left behind).
			raw := d.lines[end-1]
			line := strings.TrimSpace(raw)
			indented := strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t")
			if line == "" || (strings.HasPrefix(line, "#") && !indented) {
				end--
			} else {
				break
			}
		}
		d.fieldEnds[k] = end
		var target any
		switch k {
		case "id":
			target = &d.Task.ID
		case "title":
			target = &d.Task.Title
		case "state":
			target = &d.Task.State
		case "status":
			// Before 6ephqk step 4 (old-layout): read so the board still
			// shows, and hug upgrade can move it to state:. A state: key
			// anywhere wins.
			if hasKey(mapping, "state") {
				continue
			}
			target = &d.Task.State
		case "stage":
			target = &d.Task.Stage
		case "type":
			target = &d.Task.Type
		case "priority":
			target = &d.Task.Priority
		case "assignee":
			target = &d.Task.Assignee
		case "model":
			target = &d.Task.Model
		case "effort":
			target = &d.Task.Effort
		case "branch":
			target = &d.Task.Branch
		case "worktree":
			target = &d.Task.Worktree
		case "labels":
			target = &d.Task.Labels
		case "parent":
			target = &d.Task.Parent
		case "aliases":
			target = &d.Task.Aliases
		case "created":
			target = &d.Task.Created
		case "updated":
			target = &d.Task.Updated
		default:
			if _, node := NodeFields()[k]; !node {
				continue
			}
			var v string
			if e := value.Decode(&v); e != nil || value.Kind != yaml.ScalarNode {
				d.add(key.Line+1, fmt.Sprintf("invalid %s: a node field is one line of text", k))
				continue
			}
			if d.Task.Fields == nil {
				d.Task.Fields = map[string]string{}
			}
			d.Task.Fields[k] = v
			continue
		}
		if e := value.Decode(target); e != nil {
			d.add(key.Line+1, fmt.Sprintf("invalid %s: %v", k, e))
		}
	}
}

// hasKey says whether the mapping has key k.
func hasKey(mapping *yaml.Node, k string) bool {
	for n := 0; n+1 < len(mapping.Content); n += 2 {
		if mapping.Content[n].Value == k {
			return true
		}
	}
	return false
}

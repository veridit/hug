package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Issue is one GitHub issue as returned by `gh issue list --json`.
type Issue struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	State       string `json:"state"`       // OPEN | CLOSED
	StateReason string `json:"stateReason"` // COMPLETED | NOT_PLANNED | REOPENED | ""
	URL         string `json:"url"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
}

// FetchIssues lists every issue (open and closed) of repo ("owner/name", or
// "" for the current directory's repository) with the gh CLI, which carries
// the user's GitHub credentials. hug stores no token.
func FetchIssues(ctx context.Context, dir, repo string) ([]Issue, error) {
	args := []string{"issue", "list", "--state", "all", "--limit", "5000",
		"--json", "number,title,body,state,stateReason,url,createdAt,updatedAt,labels,assignees,milestone"}
	if repo != "" {
		args = append(args, "--repo", repo)
	}
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("gh issue list failed: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("gh is required for GitHub import (https://cli.github.com): %w", err)
	}
	var issues []Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("unexpected gh output: %w", err)
	}
	return issues, nil
}

var checkbox = regexp.MustCompile(`^\s*[-*]\s+\[([ xX])\]\s+(.+?)\s*$`)

// WriteIssues renders issues as backlog-md task files under dir/tasks, so the
// backlog importer (aliases, idempotent re-runs, local edits win) applies
// unchanged. Task-list checkboxes become acceptance items; the rest of the
// body becomes the description, which the importer turns into the goal.
func WriteIssues(dir string, issues []Issue) error {
	tasks := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasks, 0o700); err != nil {
		return err
	}
	for _, is := range issues {
		status := "To Do"
		if is.State == "CLOSED" {
			status = "Done"
			if is.StateReason == "NOT_PLANNED" {
				status = "Dropped"
			}
		}
		var labels, assignees []string
		for _, l := range is.Labels {
			labels = append(labels, l.Name)
		}
		for _, a := range is.Assignees {
			assignees = append(assignees, a.Login)
		}
		var desc, checks []string
		for _, line := range strings.Split(strings.ReplaceAll(is.Body, "\r\n", "\n"), "\n") {
			if m := checkbox.FindStringSubmatch(line); m != nil {
				mark := " "
				if m[1] != " " {
					mark = "x"
				}
				checks = append(checks, fmt.Sprintf("- [%s] #%d %s", mark, len(checks)+1, m[2]))
				continue
			}
			desc = append(desc, line)
		}
		description := strings.TrimSpace(strings.Join(desc, "\n"))
		if description == "" {
			description = is.Title
		}
		description += "\n\nGitHub: " + is.URL
		var b strings.Builder
		b.WriteString("---\n")
		fmt.Fprintf(&b, "id: GH-%d\n", is.Number)
		fmt.Fprintf(&b, "title: %s\n", yamlString(is.Title))
		fmt.Fprintf(&b, "status: %s\n", status)
		fmt.Fprintf(&b, "assignee: %s\n", yamlList(assignees))
		fmt.Fprintf(&b, "labels: %s\n", yamlList(labels))
		fmt.Fprintf(&b, "created_date: %s\n", yamlString(is.CreatedAt))
		fmt.Fprintf(&b, "updated_date: %s\n", yamlString(is.UpdatedAt))
		if is.Milestone != nil && is.Milestone.Title != "" {
			fmt.Fprintf(&b, "milestone: %s\n", yamlString(is.Milestone.Title))
		}
		b.WriteString("---\n\n## Description\n<!-- SECTION:DESCRIPTION:BEGIN -->\n")
		b.WriteString(description)
		b.WriteString("\n<!-- SECTION:DESCRIPTION:END -->\n")
		if len(checks) == 0 {
			// Every huginn task needs an observable check; most issues have
			// no task list (28 of 42 in statisticsnorway/statbus).
			mark := " "
			if status == "Done" {
				mark = "x"
			}
			checks = append(checks, fmt.Sprintf("- [%s] #1 GitHub issue #%d is resolved as described", mark, is.Number))
		}
		b.WriteString("\n## Acceptance Criteria\n<!-- AC:BEGIN -->\n")
		b.WriteString(strings.Join(checks, "\n"))
		b.WriteString("\n<!-- AC:END -->\n")
		name := fmt.Sprintf("gh-%d - %s.md", is.Number, fileSafe(is.Title))
		if err := os.WriteFile(filepath.Join(tasks, name), []byte(b.String()), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func yamlString(s string) string {
	b, _ := json.Marshal(s) // JSON strings are valid YAML double-quoted scalars
	return string(b)
}

func yamlList(xs []string) string {
	if len(xs) == 0 {
		return "[]"
	}
	q := make([]string, len(xs))
	for i, x := range xs {
		q[i] = yamlString(x)
	}
	return "[" + strings.Join(q, ", ") + "]"
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9 ._-]+`)

func fileSafe(s string) string {
	s = strings.TrimSpace(unsafeName.ReplaceAllString(s, " "))
	if len(s) > 60 {
		s = s[:60]
	}
	return s
}

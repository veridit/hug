package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// hug task models (jr55s8): the efforts each model runs at, in hug.toml
// [models] on the board branch, which the commit-msg gate checks a commit's
// Model: <id>/<effort> trailer against. Without arguments it lists them;
// "add ID LEVEL..." writes the model's line (replacing one already there)
// and commits it on the task branch.

// runModels is hug task models [add ID LEVEL...].
func (a *App) runModels(ctx context.Context, s *session, f flags) (Reply, error) {
	root := s.r.Dir
	if len(f.args) == 0 {
		ms, e := task.Models(root)
		if e != nil {
			return nil, e
		}
		next := "hug task next"
		if len(ms) == 0 {
			next = "hug task models add MODEL-ID " + strings.Join(task.EffortWords[2:5], " ")
		}
		return Reply{"models": ms, "next": next}, nil
	}
	if f.args[0] != "add" || len(f.args) < 3 {
		return nil, f.wrongCount("models takes no arguments, or add ID LEVEL...")
	}
	id := strings.TrimSpace(f.args[1])
	if id == "" || strings.ContainsAny(id, "/ \t\"[]=") || id == "human" {
		return nil, usage("a model id is one word without a slash, as the Model: trailer names it before /effort: "+id, "hug task models add claude-opus-5-5 low medium high max")
	}
	var levels []string
	for _, l := range f.args[2:] {
		l = strings.TrimSpace(l)
		if l == "" || strings.ContainsAny(l, "/ \t\"[],=") {
			return nil, usage("an effort level is one word: "+l, "hug task models add "+id+" low medium high")
		}
		levels = append(levels, l)
	}
	path := filepath.Join(root, task.ConfigName)
	b, e := os.ReadFile(path)
	if e != nil && !os.IsNotExist(e) {
		return nil, e
	}
	if e = os.WriteFile(path, []byte(task.WithModel(string(b), task.Model{ID: id, Levels: levels})), 0o644); e != nil {
		return nil, e
	}
	if e = s.commit("hug task models add " + id); e != nil {
		return nil, e
	}
	ms, _ := task.Models(root)
	return Reply{"models": ms, "changed": true, "next": "hug task models"}, nil
}

// modelsStatus is the hug setup step (crab 20:01): commits name an effort,
// checked per model once [models] lists them.
func modelsStatus(board string) (done bool, detail, do string) {
	ms, _ := task.Models(board)
	if len(ms) == 0 {
		return false, "commits name an effort (Model: <model-id>/<effort>); list each model's levels in hug.toml [models], until then any of " + strings.Join(task.EffortWords, ", ") + " goes", "hug task models add MODEL-ID " + strings.Join(task.EffortWords[2:5], " ")
	}
	var ids []string
	for _, m := range ms {
		ids = append(ids, m.ID)
	}
	return true, "commits name an effort, checked per model in hug.toml [models]: " + strings.Join(ids, ", "), ""
}

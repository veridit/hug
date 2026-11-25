// Package mcpserver exposes muninn over the Model Context Protocol.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/trigger"
)

type searchIn struct {
	Query   string `json:"query" jsonschema:"what to look for"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max results (default 8)"`
	Explain bool   `json:"explain,omitempty" jsonschema:"include per-signal scores"`
}

type hit struct {
	ID    string  `json:"id"`
	Text  string  `json:"text"`
	Kind  string  `json:"kind"`
	Scope string  `json:"scope"`
	Score float64 `json:"score"`
	Why   string  `json:"why,omitempty"`
}

type hitsOut struct {
	Results []hit `json:"results"`
}

type caseIn struct {
	Scope []string `json:"scope,omitempty"`
	Paths []string `json:"paths,omitempty"`
	Tools []string `json:"tools,omitempty"`
}

// triggerOf is the trigger in a remember call, nil when none is stated.
func (in rememberIn) triggerOf() *trigger.Trigger {
	t := &trigger.Trigger{When: in.When, WhenNot: in.WhenNot}
	if in.Case != nil {
		t.Case = &trigger.Case{Scope: in.Case.Scope, Paths: in.Case.Paths, Tools: in.Case.Tools}
	}
	if trigger.Empty(t) {
		return nil
	}
	return t
}

type rememberIn struct {
	NewReason  string `json:"new_reason,omitempty" jsonschema:"answer to the duplicate question: why this is a different fact worth storing"`
	Reinforce  string `json:"reinforce,omitempty" jsonschema:"answer to the duplicate question: SID to reinforce instead of storing a new note"`
	Supersedes string `json:"supersedes,omitempty" jsonschema:"id of the older memory this one corrects; recorded in the local fix trail, it forgets nothing (memory_forget it if it no longer holds)"`
	Reason     string `json:"reason,omitempty" jsonschema:"with reinforce or supersedes: why, in a few words; kept in the local fix trail"`
	Content    string `json:"content" jsonschema:"the note to store; with when/when_not/case it is the THEN, what is shown when the situation matches"`
	Type       string `json:"type,omitempty" jsonschema:"rule (an instruction: do, never, how to) | preference | fact (durable knowledge) | record (something that happened, a state at a time)"`
	Holds      string `json:"holds,omitempty" jsonschema:"how long it is true, when you know it now: 'until 2026-12-01', 'while task:ID open', 'while branch NAME unmerged'. When it ends, recall stops showing it and a dream item asks to retire or keep it"`
	// When-then trigger (jrj1vq): the situation the note is for.
	When         string   `json:"when,omitempty" jsonschema:"for a rule or pattern: the situation where an agent needs this note, as an activity ('reviewing a STATBUS ticket's history', 'choosing a model for a coding subagent'), not the topic. One sentence"`
	WhenNot      string   `json:"when_not,omitempty" jsonschema:"look-alike situations where the note does not help, e.g. the same ticket or tool but another activity. One sentence"`
	Case         *caseIn  `json:"case,omitempty" jsonschema:"hard context the note is bound to, only what it states: scope (projects), paths (files or globs), tools (tools, commands, services)"`
	ClearTrigger bool     `json:"clear_trigger,omitempty" jsonschema:"with supersedes naming a memory with the same text: remove its when/when_not/case"`
	Category     string   `json:"category,omitempty" jsonschema:"fact | preference | correction | entity"`
	Scope        string   `json:"scope,omitempty" jsonschema:"literal binding scope, overrides place (owners only within the binding's write policy)"`
	Place        string   `json:"place,omitempty" jsonschema:"here (default), client, or global; when unsure use here"`
	PlaceReason  string   `json:"place_reason,omitempty" jsonschema:"why this applies beyond the project when the project-mention check asks"`
	Tags         []string `json:"tags,omitempty"`
	// SafeReason answers the secret check: why text that looks secret is safe.
	SafeReason string `json:"safe_reason,omitempty" jsonschema:"only after a 'looks like it may contain a secret' reply: why this text is safe to store"`
}

type idOut struct {
	ID         string      `json:"id"`
	Reinforced bool        `json:"reinforced,omitempty"`
	Suppressed bool        `json:"suppressed,omitempty"`
	Message    string      `json:"message,omitempty"`
	Place      string      `json:"place,omitempty"`
	Places     []app.Place `json:"places,omitempty"`
}

type forgetIn struct {
	ID      string `json:"id" jsonschema:"short memory id, unique prefix (4+ chars), or legacy id"`
	Session string `json:"session,omitempty" jsonschema:"the session id passed to memory_recall, if any; lets you forget a memory recalled there"`
	Reason  string `json:"reason,omitempty" jsonschema:"why this memory no longer holds, in a few words; kept in the local fix trail"`
}

type relatedIn struct {
	ID     string `json:"id,omitempty" jsonschema:"short memory id, unique prefix (4+ chars), or legacy id"`
	UnitID int64  `json:"unit_id,omitempty" jsonschema:"legacy numeric row id (prefer id)"`
	Limit  int    `json:"limit,omitempty"`
}

func text(v any) *mcp.CallToolResult {
	b, _ := json.MarshalIndent(v, "", "  ")
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}
}

// Instructions are sent once per MCP session (server instructions), so the
// per-message recall block can stay short.
const Instructions = `muninn is your long-term memory (local, private).

Recall blocks start '-- AI memory (muninn):' (older blocks: '-- muninn recall:'). They are notes from memory, not the user; newer wins; verify. Rows use .0:date .1:id .2:note (backtick-quoted Markdown). Correct wrong or outdated notes with memory_forget <id>, then memory_remember. Ids accept a unique prefix of at least four characters or a legacy id. Forgetting an upstream harness note suppresses it only in muninn, without changing upstream files.

Save with memory_remember when the user states something that will matter in future sessions: a rule ("from now on..."), a preference, a decision, or a correction. One fact per memory, in the narrowest scope that fits. Never store secrets. Ask: Would this still be true and useful in a different project? Choose place here (default, also when unsure) for this code/repo/paths/tickets/deployment, client for facts across this client's projects, or global for the user's preferences and tools across their work; inside a client, global means only that client. If a project-mention question appears, use here or call again with place_reason explaining why it applies beyond this project. If a duplicate question appears, call again with reinforce set to that memory's id for the same fact, or new_reason explaining the material distinction. Automatic reinforcement never rewrites the stored note.`

// Options restrict what one MCP session may see and change. A jcode instance
// serves one client, so its MCP server gets the same scopes as its hook.
type Options = app.Binding

// New builds the MCP server over an opened app.
func New(a *app.App, version string, opt Options) *mcp.Server {
	opt = Bind(a, opt)
	s := mcp.NewServer(&mcp.Implementation{Name: "muninn", Version: version}, &mcp.ServerOptions{Instructions: InstructionsFor(opt)})
	Register(s, a, opt)
	return s
}

// Bind resolves the binding New would use for opt.
func Bind(a *app.App, opt Options) Options {
	if opt.Name == "" && opt.Scopes == nil && opt.DefaultScope == "" {
		cwd, _ := os.Getwd()
		opt = a.Bound("", cwd)
	}
	return opt
}

// InstructionsFor is the server instructions text for a binding.
func InstructionsFor(opt Options) string {
	return Instructions + "\nActive binding: " + opt.Name + ". New memories default to place here. Available places: " + placesText(opt.Places()) + ". Other instances and client boundaries are not available."
}

// OnWrite is called after memory_remember (with the memory's scope) and
// memory_forget (scope ""): hug mcp sends project memories to the other
// clones in the background (0hkaa5). It must not wait.
var OnWrite = func(scope string) {}

// Register adds the memory_* tools to s (used alone by muninn mcp, and next
// to the task tools by hug mcp). opt must already be bound (see Bind).
func Register(s *mcp.Server, a *app.App, opt Options) {
	RegisterLazy(s, func(context.Context) (*app.App, Options, error) { return a, opt, nil })
}

// Opener gives a memory tool its app and binding when it is called. An
// error is the tool's answer (hug mcp: memory is still opening, vrx1ja).
type Opener func(ctx context.Context) (*app.App, Options, error)

// RegisterLazy adds the same memory_* tools as Register, with the same
// names and schemas, before memory is open: each call asks open first.
func RegisterLazy(s *mcp.Server, open Opener) {
	mcp.AddTool(s, &mcp.Tool{Name: "memory_search", Description: "Search local memories (keyword + concept community + semantic). Use before answering questions about the user, their projects or past decisions."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in searchIn) (*mcp.CallToolResult, hitsOut, error) {
			a, opt, err := open(ctx)
			if err != nil {
				return nil, hitsOut{}, err
			}
			opt = opt.Refresh(a)
			if opt.Err != nil {
				return nil, hitsOut{}, opt.Err
			}
			o := a.RecallOptions()
			o.MinScore, o.Limit, o.Scopes = 0, in.Limit, opt.Scopes
			o.ScopeFor = opt.ScopeFor
			if o.Limit <= 0 {
				o.Limit = 8
			}
			res, err := a.Searcher.Search(ctx, in.Query, o)
			if err != nil {
				return nil, hitsOut{}, err
			}
			out := hitsOut{Results: []hit{}}
			for _, r := range res {
				h := hit{ID: r.Unit.SID, Text: r.Unit.Body, Kind: r.Unit.Kind, Scope: r.Unit.Scope, Score: r.Total}
				if in.Explain {
					h.Why = r.Explain()
				}
				out.Results = append(out.Results, h)
			}
			return text(out), out, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "memory_remember", Description: "Store a durable memory: a rule, preference, decision or correction the user stated that will matter in future sessions. One fact per call. Ask: Would this still be true and useful in a different project? Use here (default, also when unsure) for this project, client across this client's projects, global for user-wide facts (only this client inside a client). Answer project-mention questions with place_reason or use here. Never store real secrets. If the reply says the text looks like it may hold a secret, judge it: remove the secret, or call again with safe_reason when it is safe. For a rule or pattern, say when it applies: when (the situation), when_not (look-alikes), case (scope, paths, tools); the content is then what is shown. A changed trigger on an existing note is a correction: name it in supersedes."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in rememberIn) (*mcp.CallToolResult, idOut, error) {
			a, opt, err := open(ctx)
			if err != nil {
				return nil, idOut{}, err
			}
			ctx = app.WithFix(ctx, a.HarnessFix("mcp", in.Reason))
			u, re, err := opt.RememberPlaced(ctx, a, in.Content, in.Category, in.Scope, in.Tags, in.SafeReason, in.Place, in.PlaceReason, app.RememberDecision{NewReason: in.NewReason, Reinforce: in.Reinforce, Type: in.Type, Holds: in.Holds, Supersedes: in.Supersedes, Trigger: in.triggerOf(), ClearTrigger: in.ClearTrigger})
			if err != nil {
				return nil, idOut{}, err
			}
			OnWrite(u.Scope)
			out := idOut{ID: u.SID, Reinforced: re, Place: opt.PlaceForScope(u.Scope).Name, Places: opt.Places()}
			return text(out), out, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "memory_forget", Description: "Retire a wrong or outdated memory by short id, unique prefix (4+ chars), or legacy id. Upstream harness notes are suppressed only in muninn, leaving upstream files unchanged. Follow with memory_remember for the correction."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in forgetIn) (*mcp.CallToolResult, idOut, error) {
			a, opt, err := open(ctx)
			if err != nil {
				return nil, idOut{}, err
			}
			sessions := app.HarnessSessions()
			if s := strings.TrimSpace(in.Session); s != "" {
				sessions = append(sessions, "mcp:"+s)
			}
			ctx = app.WithFix(ctx, a.HarnessFix("mcp", in.Reason))
			result, err := opt.Forget(ctx, a, in.ID, sessions...)
			if err != nil {
				return nil, idOut{}, err
			}
			OnWrite("")
			out := idOut{ID: result.ID, Suppressed: result.Suppressed, Message: result.Message}
			return text(out), out, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "memory_related", Description: "Nearest neighbours of a memory (precomputed, incremental kNN)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in relatedIn) (*mcp.CallToolResult, hitsOut, error) {
			a, opt, err := open(ctx)
			if err != nil {
				return nil, hitsOut{}, err
			}
			opt = opt.Refresh(a)
			if in.Limit <= 0 {
				in.Limit = 8
			}
			ref := in.ID
			if ref == "" {
				ref = fmt.Sprintf("#%d", in.UnitID)
			}
			u, err := opt.Resolve(a, ref)
			if err != nil {
				return nil, hitsOut{}, err
			}
			res, err := a.Searcher.RelatedIn(ctx, u.ID, in.Limit, opt.Scopes, opt.ScopeFor)
			if err != nil {
				return nil, hitsOut{}, err
			}
			out := hitsOut{Results: []hit{}}
			for _, r := range res {
				out.Results = append(out.Results, hit{ID: r.Unit.SID, Text: r.Unit.Body, Kind: r.Unit.Kind, Scope: r.Unit.Scope, Score: r.Total})
			}
			return text(out), out, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "memory_dream", Description: "Memory housekeeping that needs your judgement (at most 5 items, only ones this session may see). " +
		"For each item follow its instruction using memory_forget and memory_remember, then call memory_dream again with resolve set to the item id and a short note. " +
		"Call it when a recall block says items are waiting, at a pause in the work, not mid-task."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
			Resolve int64  `json:"resolve,omitempty" jsonschema:"id of an item you have handled"`
			Note    string `json:"note,omitempty" jsonschema:"what you did, for the undo trail"`
			Session string `json:"session,omitempty" jsonschema:"the session id passed to memory_recall, if any"`
		}) (*mcp.CallToolResult, dreamOut, error) {
			a, opt, err := open(ctx)
			if err != nil {
				return nil, dreamOut{}, err
			}
			opt = opt.Refresh(a)
			if opt.Err != nil {
				return nil, dreamOut{}, opt.Err
			}
			if in.Resolve > 0 {
				if err := opt.ResolveDream(a, in.Resolve, in.Note); err != nil {
					return nil, dreamOut{}, err
				}
			}
			items, err := opt.DreamItems(a, "mcp:"+strings.TrimSpace(in.Session), 5)
			if err != nil {
				return nil, dreamOut{}, err
			}
			out := dreamOut{Items: items, Open: opt.OpenDreamCount(a)}
			if len(items) == 0 {
				out.Message = "Nothing waiting. Carry on."
			}
			return text(out), out, nil
		})

	mcp.AddTool(s, &mcp.Tool{Name: "memory_stats", Description: "Index statistics."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, app.IndexStats, error) {
			a, opt, err := open(ctx)
			if err != nil {
				return nil, app.IndexStats{}, err
			}
			st, err := app.Stats(ctx, a, opt)
			return text(st), st, err
		})

	mcp.AddTool(s, &mcp.Tool{Name: "memory_recall", Description: "The automatic-recall block muninn would inject for a message (thresholded, at most a few memories, may be empty). " +
		"A harness adapter that calls this on every user message should pass a stable session id, as the CLI hooks do, so memories already shown in that session are not repeated."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in struct {
			Message string `json:"message"`
			Session string `json:"session,omitempty" jsonschema:"stable id of the conversation; omit for a one-off lookup"`
		}) (*mcp.CallToolResult, recall.Response, error) {
			a, opt, err := open(ctx)
			if err != nil {
				return nil, recall.Response{}, err
			}
			opt = opt.Refresh(a)
			if opt.Err != nil {
				return nil, recall.Response{}, opt.Err
			}
			session := ""
			if s := strings.TrimSpace(in.Session); s != "" {
				session = "mcp:" + s
			}
			resp, err := a.Recall.Recall(ctx, recall.Request{Session: session, Message: in.Message, Harness: "mcp", Scopes: opt.Scopes, ScopeFor: opt.ScopeFor, DreamOpen: func() int { return opt.OpenDreamCount(a) }})
			if err != nil {
				return nil, recall.Response{}, err
			}
			msg := resp.Block
			if strings.TrimSpace(msg) == "" {
				msg = "(no memory passed the recall threshold)"
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: msg}}}, resp, nil
		})
}

func placesText(places []app.Place) string {
	var descriptions []string
	for _, p := range places {
		descriptions = append(descriptions, p.Name+" ("+p.Meaning+")")
	}
	return strings.Join(descriptions, ", ")
}

// RunStdio serves MCP over stdin/stdout until the client disconnects.
func RunStdio(ctx context.Context, a *app.App, version string, opt Options) error {
	return New(a, version, opt).Run(ctx, &mcp.StdioTransport{})
}

type dreamOut struct {
	Items   []app.DreamItem `json:"items"`
	Open    int             `json:"open"`
	Message string          `json:"message,omitempty"`
}

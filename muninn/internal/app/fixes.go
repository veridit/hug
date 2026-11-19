package app

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/store"
)

// Memory fixes (rjk44w): every forget, every remember that names an old
// memory (reinforce, supersedes), and every rescope (j13xs0: one memory, or
// each memory of a --from batch) leaves one row in memory_fixes, so we can
// learn whether agents act on what recall shows them and whether a forget
// was later undone. Local and free: no model call, nothing recall reads.
// Recording never blocks the write it describes: its errors are dropped.

// Fix is one row of the trail.
type Fix struct {
	At       time.Time `json:"at"`
	Action   string    `json:"action"` // forget | reinforce | supersede | rescope
	SID      string    `json:"sid"`    // the memory fixed
	NewSID   string    `json:"new_sid,omitempty"`
	Session  string    `json:"session,omitempty"` // harness:session, first one given
	Harness  string    `json:"harness,omitempty"`
	Instance string    `json:"instance,omitempty"` // jcode instance name, when known
	Surface  string    `json:"surface,omitempty"`  // cli | mcp | web | tui
	Reason   string    `json:"reason,omitempty"`
	// Shown: recall showed this memory to one of the caller's sessions
	// before the fix; ShownAgo is how long before (the latest showing).
	Shown       bool          `json:"shown"`
	ShownAgo    time.Duration `json:"shown_ago_ns,omitempty"`
	ContentHash string        `json:"content_hash,omitempty"`
	// A rescope: the scope the memory left and the one it moved to, and
	// how (single: one memory, batch: hug mem rescope --from).
	OldScope string `json:"old_scope,omitempty"`
	NewScope string `json:"new_scope,omitempty"`
	Mode     string `json:"mode,omitempty"`
}

// FixContext is who is fixing, carried on the context from the surface (CLI
// flags, MCP arguments) down to the write.
type FixContext struct {
	Sessions          []string // harness:session keys, as recall records them
	Harness, Instance string
	Surface, Reason   string
	Now               time.Time // zero: time.Now (tests set it)
}

type fixKey struct{}

// WithFix carries fc on ctx for the forget or remember it calls.
func WithFix(ctx context.Context, fc FixContext) context.Context {
	return context.WithValue(ctx, fixKey{}, fc)
}

func fixFrom(ctx context.Context) FixContext {
	fc, _ := ctx.Value(fixKey{}).(FixContext)
	return fc
}

// HarnessFix is the fix context of a process a harness started: its
// sessions (HarnessSessions), the harness of the first one, and for jcode
// the instance its JCODE_HOME belongs to.
func (a *App) HarnessFix(surface, reason string) FixContext {
	fc := FixContext{Sessions: HarnessSessions(), Surface: surface, Reason: reason}
	sort.Strings(fc.Sessions) // map order in HarnessSessions: keep it stable
	if len(fc.Sessions) > 0 {
		fc.Harness = fc.Sessions[0][:strings.IndexByte(fc.Sessions[0], ':')]
	}
	if os.Getenv("JCODE_SESSION_ID") != "" {
		h := os.Getenv("JCODE_HOME")
		if h == "" {
			if home, err := os.UserHomeDir(); err == nil {
				h = filepath.Join(home, ".jcode")
			}
		}
		if h != "" {
			fc.Instance = harness.InstanceForHome(a.JcodeInstances(), h).Name
		}
	}
	return fc
}

// recordFixes writes one row per memory unit fixed, for the caller in ctx.
// units are the memories as they were before the write. Errors are dropped:
// a trail row is never worth a failed forget.
func recordFixes(ctx context.Context, a *App, action string, units []fixTarget) {
	fc := fixFrom(ctx)
	now := fc.Now
	if now.IsZero() {
		now = time.Now()
	}
	session := ""
	for _, s := range fc.Sessions {
		if s != "" && !strings.HasSuffix(s, ":") {
			session = s
			break
		}
	}
	harness := fc.Harness
	if harness == "" && session != "" {
		harness = session[:strings.IndexByte(session, ':')]
	}
	for _, u := range units {
		f := Fix{At: now, Action: action, SID: u.sid, NewSID: u.newSID, Session: session, Harness: harness, Instance: fc.Instance, Surface: fc.Surface, Reason: strings.TrimSpace(fc.Reason), ContentHash: u.hash, OldScope: u.oldScope, NewScope: u.newScope, Mode: u.mode}
		if at, ok := a.lastShown(u.ids, fc.Sessions); ok {
			f.Shown = true
			if ago := now.Sub(at); ago > 0 {
				f.ShownAgo = ago
			}
		}
		_ = RecordFix(ctx, a, f)
	}
}

// fixTarget is one memory a fix names: its sid, every unit row it has (a
// memory may have twins across sources), and its content hash.
type fixTarget struct {
	sid, newSID, hash        string
	ids                      []int64
	oldScope, newScope, mode string // rescope only
}

// rescopeTarget is the trail target for a rescope of u to scope. Only u's
// own row is looked up for "shown": a batch may name thousands.
func rescopeTarget(u store.Unit, scope, mode string) fixTarget {
	return fixTarget{sid: u.SID, hash: u.ContentHash, ids: []int64{u.ID}, oldScope: u.Scope, newScope: scope, mode: mode}
}

// lastShown is when recall last showed any of ids to one of sessions.
func (a *App) lastShown(ids []int64, sessions []string) (time.Time, bool) {
	var best int64
	for _, s := range sessions {
		if s == "" || strings.HasSuffix(s, ":") {
			continue
		}
		for _, id := range ids {
			var at sql.NullInt64
			if a.DB.QueryRow(`SELECT MAX(at_ns) FROM injections WHERE session=? AND unit_id=?`, s, id).Scan(&at) == nil && at.Valid && at.Int64 > best {
				best = at.Int64
			}
		}
	}
	if best == 0 {
		return time.Time{}, false
	}
	return time.Unix(0, best), true
}

// RecordFix stores one trail row as given.
func RecordFix(ctx context.Context, a *App, f Fix) error {
	if f.At.IsZero() {
		f.At = time.Now()
	}
	shown := 0
	if f.Shown {
		shown = 1
	}
	_, err := a.DB.ExecContext(ctx, `INSERT INTO memory_fixes(at_ns,action,sid,new_sid,session,harness,instance,surface,reason,shown,shown_ago_ns,content_hash,old_scope,new_scope,mode) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		f.At.UnixNano(), f.Action, f.SID, f.NewSID, f.Session, f.Harness, f.Instance, f.Surface, f.Reason, shown, int64(f.ShownAgo), f.ContentHash, f.OldScope, f.NewScope, f.Mode)
	return err
}

// Fixes lists trail rows at or after since, oldest first.
func Fixes(ctx context.Context, a *App, since time.Time) ([]Fix, error) {
	rows, err := a.DB.QueryContext(ctx, `SELECT at_ns,action,sid,new_sid,session,harness,instance,surface,reason,shown,shown_ago_ns,content_hash,old_scope,new_scope,mode FROM memory_fixes WHERE at_ns>=? ORDER BY at_ns, rowid`, since.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Fix
	for rows.Next() {
		var f Fix
		var at, ago int64
		var shown int
		if err := rows.Scan(&at, &f.Action, &f.SID, &f.NewSID, &f.Session, &f.Harness, &f.Instance, &f.Surface, &f.Reason, &shown, &ago, &f.ContentHash, &f.OldScope, &f.NewScope, &f.Mode); err != nil {
			return nil, err
		}
		f.At, f.Shown, f.ShownAgo = time.Unix(0, at).UTC(), shown == 1, time.Duration(ago)
		out = append(out, f)
	}
	return out, rows.Err()
}

// UndoWindow: a forget whose memory is named again (reinforced or
// superseded) within this long counts as undone.
const UndoWindow = 7 * 24 * time.Hour

// FixDay is one UTC day of the summary.
type FixDay struct {
	Day        string `json:"day"`
	Blocks     int    `json:"recall_blocks"` // distinct (session, second) recall injections
	Fixes      int    `json:"fixes"`
	OfShown    int    `json:"of_shown"`
	NotShown   int    `json:"not_shown"`
	Forgets    int    `json:"forget"`
	Reinforces int    `json:"reinforce"`
	Supersedes int    `json:"supersede"`
	Undone     int    `json:"undone"`
	Rescopes   int    `json:"rescope"`
}

// FixInstance is the fixes per jcode instance ("" when unknown).
type FixInstance struct {
	Instance string `json:"instance"`
	Fixes    int    `json:"fixes"`
	OfShown  int    `json:"of_shown"`
}

// FixSummary is what hug mem stats --fixes prints.
type FixSummary struct {
	Since     string        `json:"since,omitempty"`
	Days      []FixDay      `json:"days"`
	Instances []FixInstance `json:"instances"`
	Recent    []Fix         `json:"recent"` // the last 20, oldest first
}

// SummarizeFixes counts the trail and the recall blocks per day since.
func SummarizeFixes(ctx context.Context, a *App, since time.Time) (FixSummary, error) {
	s := FixSummary{Days: []FixDay{}, Instances: []FixInstance{}, Recent: []Fix{}}
	if !since.IsZero() {
		s.Since = since.UTC().Format("2006-01-02")
	}
	fs, err := Fixes(ctx, a, since)
	if err != nil {
		return s, err
	}
	all, err := Fixes(ctx, a, time.Time{})
	if err != nil {
		return s, err
	}
	days := map[string]*FixDay{}
	day := func(d string) *FixDay {
		if days[d] == nil {
			days[d] = &FixDay{Day: d}
		}
		return days[d]
	}
	rows, err := a.DB.QueryContext(ctx, `SELECT session, at_ns/1000000000 FROM injections WHERE at_ns>=? GROUP BY session, at_ns/1000000000`, since.UnixNano())
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var session string
		var sec int64
		if err := rows.Scan(&session, &sec); err != nil {
			rows.Close()
			return s, err
		}
		day(time.Unix(sec, 0).UTC().Format("2006-01-02")).Blocks++
	}
	rows.Close()
	inst := map[string]*FixInstance{}
	for _, f := range fs {
		d := day(f.At.UTC().Format("2006-01-02"))
		d.Fixes++
		if f.Shown {
			d.OfShown++
		} else {
			d.NotShown++
		}
		switch f.Action {
		case "forget":
			d.Forgets++
			if undone(f, all) {
				d.Undone++
			}
		case "reinforce":
			d.Reinforces++
		case "supersede":
			d.Supersedes++
		case "rescope":
			d.Rescopes++
		}
		if inst[f.Instance] == nil {
			inst[f.Instance] = &FixInstance{Instance: f.Instance}
		}
		inst[f.Instance].Fixes++
		if f.Shown {
			inst[f.Instance].OfShown++
		}
	}
	for _, d := range days {
		s.Days = append(s.Days, *d)
	}
	sort.Slice(s.Days, func(i, j int) bool { return s.Days[i].Day < s.Days[j].Day })
	for _, i := range inst {
		s.Instances = append(s.Instances, *i)
	}
	sort.Slice(s.Instances, func(i, j int) bool {
		if s.Instances[i].Fixes != s.Instances[j].Fixes {
			return s.Instances[i].Fixes > s.Instances[j].Fixes
		}
		return s.Instances[i].Instance < s.Instances[j].Instance
	})
	if n := len(fs); n > 20 {
		fs = fs[n-20:]
	}
	s.Recent = append(s.Recent, fs...)
	return s, nil
}

// undone: a later reinforce or supersede names the forgotten memory within
// UndoWindow. A rescope moves a memory; it does not bring one back.
func undone(f Fix, all []Fix) bool {
	for _, g := range all {
		if g.SID == f.SID && (g.Action == "reinforce" || g.Action == "supersede") && g.At.After(f.At) && g.At.Sub(f.At) <= UndoWindow {
			return true
		}
	}
	return false
}

// FixReport is the text form of a summary.
func FixReport(s FixSummary) string {
	var b strings.Builder
	if s.Since != "" {
		fmt.Fprintf(&b, "memory fixes since %s (UTC days)\n", s.Since)
	} else {
		b.WriteString("memory fixes (UTC days)\n")
	}
	if len(s.Days) == 0 {
		b.WriteString("no recall blocks and no fixes recorded\n")
		return b.String()
	}
	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "day\tblocks\tfixes\tof-shown\tnot-shown\tforget\treinforce\tsupersede\tundone\trescope")
	for _, d := range s.Days {
		fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", d.Day, d.Blocks, d.Fixes, d.OfShown, d.NotShown, d.Forgets, d.Reinforces, d.Supersedes, d.Undone, d.Rescopes)
	}
	w.Flush()
	if len(s.Instances) > 0 {
		b.WriteString("by instance:")
		for _, i := range s.Instances {
			name := i.Instance
			if name == "" {
				name = "unknown"
			}
			fmt.Fprintf(&b, " %s %d (%d of shown)", name, i.Fixes, i.OfShown)
		}
		b.WriteString("\n")
	}
	if len(s.Recent) > 0 {
		b.WriteString("recent:\n")
		for _, f := range s.Recent {
			shown := "not shown"
			if f.Shown {
				shown = "shown " + roundAgo(f.ShownAgo) + " before"
			}
			line := fmt.Sprintf("  %s %s %s", f.At.UTC().Format("2006-01-02 15:04"), f.Action, f.SID)
			if f.NewSID != "" {
				line += " -> " + f.NewSID
			}
			if f.Action == "rescope" {
				line += " " + f.OldScope + " -> " + f.NewScope
				if f.Mode != "" {
					line += " (" + f.Mode + ")"
				}
			}
			line += " " + shown
			for _, part := range []struct{ k, v string }{{"via", f.Surface}, {"instance", f.Instance}, {"session", f.Session}} {
				if part.v != "" {
					line += " " + part.k + " " + part.v
				}
			}
			b.WriteString(line + "\n")
			if f.Reason != "" {
				b.WriteString("    reason: " + f.Reason + "\n")
			}
		}
	}
	return b.String()
}

// roundAgo prints a duration to a readable unit: 45s, 3m, 2h, 4d.
func roundAgo(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

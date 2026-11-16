package app

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/agents"
)

// Memories that name an ephemeral agent (ez50rb). Agents come and go within
// hours; "report to coordinator orangutan" means nothing once orangutan has
// stopped. Each such memory becomes one dream item of kind "agent", naming
// the tickets that agent's sessions worked on when they can be found, so
// the agent judging it can rewrite the memory to the ticket (or file,
// commit, model) instead. One at a time through dream, never a purge.

// AgentPerPass bounds new agent items per dream pass.
var AgentPerPass = 10

// AgentIdle: an agent whose session files are older than this has stopped.
var AgentIdle = 24 * time.Hour

const agentInstruction = "This memory names %s, a swarm agent: agents stop within hours, so the name will not mean anything later.%s " +
	"Rewrite it to name what it is about (the ticket, file, commit or model) with memory_remember, then memory_forget this one. " +
	"If it is only about that agent's status or coordination, memory_forget it. If it is a lasting fact that needs the name, leave it (resolve with note kept)."

// ticketRef finds ticket ids in a session file's first messages: board ids
// in hug task commands and TASK-N style ids.
var ticketRef = regexp.MustCompile(`hug task (?:take|show) ([a-z0-9]{6})\b|\b(TASK-\d+(?:\.\d+)?|[A-Z][a-z]+-\d{3,5})\b`)

// citation: "Odersky-2004" is a paper, not a ticket (seen on the store).
var citation = regexp.MustCompile(`^[A-Z][a-z]+-(?:19|20)\d\d$`)

// sessionTickets lists up to 3 tickets the given sessions worked on (most
// mentioned first), read from the first 256 KiB of each session file.
func sessionTickets(ss []agents.Session) []string {
	n := map[string]int{}
	for _, s := range ss {
		fh, err := os.Open(s.Path)
		if err != nil {
			continue
		}
		buf := make([]byte, 256<<10)
		k, _ := io.ReadFull(fh, buf)
		fh.Close()
		for _, m := range ticketRef.FindAllStringSubmatch(string(buf[:k]), -1) {
			id := m[1]
			if id == "" {
				id = m[2]
			}
			if citation.MatchString(id) {
				continue
			}
			n[id]++
		}
	}
	ids := make([]string, 0, len(n))
	for id := range n {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return n[ids[i]] > n[ids[j]] || n[ids[i]] == n[ids[j]] && ids[i] < ids[j] })
	if len(ids) > 3 {
		ids = ids[:3]
	}
	return ids
}

// queueAgentNames queues memories naming agents, AgentPerPass at a time.
func (a *App) queueAgentNames(dry bool, now time.Time) (int, error) {
	var homes []string
	for _, i := range a.JcodeInstances() {
		homes = append(homes, i.Home)
	}
	sessions := agents.Sessions(homes)
	if len(sessions) == 0 {
		return 0, nil
	}
	names := make([]string, 0, len(sessions))
	for n := range sessions {
		names = append(names, n)
	}
	sort.Strings(names)
	m := agents.New(names)
	// Incremental: each pass reads only memories changed since the last
	// scan (a full scan of 29,118 memories took 3.2 s). A memory cannot name
	// an agent that did not exist when it was written, so new names need no
	// rescan of old memories.
	var since int64
	if v := a.DB.GetMeta("agent_scan_ns"); v != "" {
		fmt.Sscan(v, &since)
	}
	rows, err := a.DB.Query(`SELECT u.sid, u.scope, u.body, max(u.updated_ns) FROM units u WHERE u.active=1 AND u.kind='memory' AND u.sid IS NOT NULL
		AND u.updated_ns > ?
		AND NOT EXISTS (SELECT 1 FROM suppressed x WHERE x.sid=u.sid)
		AND NOT EXISTS (SELECT 1 FROM dream_items d WHERE d.kind='agent' AND d.sids=u.sid)
		GROUP BY u.sid ORDER BY max(u.updated_ns)`, since)
	if err != nil {
		return 0, err
	}
	type cand struct {
		sid, scope string
		names      []string
		meant      map[string][]agents.Session
	}
	var cs []cand
	mark, pinned := since, false
	for rows.Next() {
		var c cand
		var body string
		var upd int64
		if rows.Scan(&c.sid, &c.scope, &body, &upd) != nil {
			continue
		}
		c.names = m.Find(body)
		// The agent a memory names is the session of that name alive when
		// it was written (names are reused). It has stopped when none of
		// those sessions wrote for AgentIdle.
		running := false
		c.meant = map[string][]agents.Session{}
		for _, name := range c.names {
			c.meant[name] = agents.Meant(sessions[name], time.Unix(0, upd))
			for _, s := range c.meant[name] {
				running = running || now.Sub(s.Last) < AgentIdle
			}
		}
		full := !dry && len(c.names) > 0 && len(cs) >= AgentPerPass
		if (running || full) && !pinned {
			// Not decided yet, so the next pass rescans from here: a memory
			// naming a running agent waits until it stops. Later memories
			// are still queued; the dream_items check skips them next time.
			mark, pinned = upd-1, true
		}
		if full {
			break
		}
		if !pinned {
			mark = upd
		}
		if len(c.names) > 0 && !running {
			cs = append(cs, c)
		}
	}
	rows.Close()
	if dry {
		return len(cs), nil
	}
	if mark > since {
		if err := a.DB.SetMeta("agent_scan_ns", fmt.Sprint(mark)); err != nil {
			return 0, err
		}
	}
	n := 0
	for _, c := range cs {
		var hint []string
		for _, name := range c.names {
			if t := sessionTickets(c.meant[name]); len(t) > 0 {
				hint = append(hint, name+" worked on "+strings.Join(t, ", "))
			}
		}
		h := ""
		if len(hint) > 0 {
			h = " From its sessions: " + strings.Join(hint, "; ") + "."
		}
		q, err := a.queueItem("agent", "agent:"+c.sid, c.sid, c.scope, fmt.Sprintf(agentInstruction, strings.Join(c.names, ", "), h), now)
		if err != nil {
			return n, err
		}
		if q {
			n++
		}
	}
	return n, nil
}

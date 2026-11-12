package recall

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Dream hand-out (2026-11-12, user: "find when there's a break"). Judgement
// items from "muninn dream" are offered to the agent only at a natural pause,
// never mid-task. A pause is one of:
//
//   - start:   the first user turn of a session
//   - return:  the user comes back after BreakGap of silence
//   - wrap-up: the message closes a stretch of work ("thanks", "good night")
//
// The cue is one line, at most once per session per CueEvery. Which signal
// fired is reported so the hand-out point can be tuned from real use.

// BreakGap is the silence after which a user turn counts as a return.
var BreakGap = 20 * time.Minute

// CueEvery limits dream cues per session.
var CueEvery = 6 * time.Hour

var wrapUp = regexp.MustCompile(`^\W*(thanks|thank you|thx|ty|great|perfect|nice work|good work|well done|good night|goodnight|that'?s all|that'?s it|done for (now|today)|let'?s (stop|call it|pause)|i'?ll be back|takk|god natt|ha det)\b`)

// noteTurn records this user turn and reports a break signal ("" if none).
func (e *Engine) noteTurn(session, msg string, now time.Time) string {
	if session == "" {
		return ""
	}
	var last, turns int64
	err := e.DB.QueryRow(`SELECT last_ns, turns FROM session_turns WHERE session=?`, session).Scan(&last, &turns)
	e.DB.Exec(`INSERT INTO session_turns(session,last_ns,turns) VALUES(?,?,1)
		ON CONFLICT(session) DO UPDATE SET last_ns=excluded.last_ns, turns=turns+1`, session, now.UnixNano())
	switch {
	case err != nil:
		return "start"
	case now.Sub(time.Unix(0, last)) >= BreakGap:
		return "return"
	case wrapUp.MatchString(strings.ToLower(strings.TrimSpace(msg))) && len(msg) < 200:
		return "wrap-up"
	}
	return ""
}

// dreamCue returns the one-line cue when items are open, the turn is a break,
// and this session has not been cued recently.
func (e *Engine) dreamCue(session, signal string, open int, now time.Time) string {
	if open <= 0 || signal == "" || session == "" {
		return ""
	}
	var cued int64
	e.DB.QueryRow(`SELECT dream_cued_ns FROM session_turns WHERE session=?`, session).Scan(&cued)
	if cued > 0 && now.Sub(time.Unix(0, cued)) < CueEvery {
		return ""
	}
	e.DB.Exec(`UPDATE session_turns SET dream_cued_ns=? WHERE session=?`, now.UnixNano(), session)
	items := "items need"
	if open == 1 {
		items = "item needs"
	}
	return fmt.Sprintf("Memory housekeeping (%s): %d %s your judgement. When this reply is done, call memory_dream, resolve them briefly, then carry on.", signal, open, items)
}

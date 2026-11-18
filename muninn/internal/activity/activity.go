// Package activity learns the person's weekly rhythm (vk9td0): when they
// type, per hour of the week, and so where their quiet stretches are. The
// daily dream pass starts at the beginning of the day's quiet stretch once a
// week of activity exists; before that, dream keeps its old rule (away 30
// min).
//
// The idea is the one Jørgen's syncthing fork uses to place its database
// defrag (~/Projects/syncthing internal/activityhist, patch/43 and patch/53):
// histogram the activity cyclically, trust it only after 7 days, start the
// expensive work at the beginning of the longest idle run. hug keeps the
// hour-of-week histogram only (168 buckets: days differ, and a week is the
// natural cycle) and counts a person's turns, not log lines.
//
// Storage is one row per hour that had a turn (activity_hours), at most 8
// weeks kept; the histogram is built from the last Weeks weeks on read, so a
// changed rhythm shows within a month (activityhist halves its counters
// weekly for the same reason). Local and free: no model, no network.
package activity

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/store"
)

const (
	// HoursPerWeek is the number of buckets.
	HoursPerWeek = 7 * 24
	// MinDays of activity before the histogram is trusted (activityhist's
	// minDaysForPrediction).
	MinDays = 7
	// Weeks of history the histogram looks at.
	Weeks = 4
	// keep is how long hours are stored.
	keep = 8 * 7 * 24 * time.Hour
)

// Location is the time zone hours of the week are counted in: people sleep
// in local time. Tests pin it.
var Location = func() *time.Location { return time.Local }

const seededKey = "activity_seeded"

// HourOfWeek is t's bucket: Monday 00:00 is 0, Sunday 23:00 is 167.
func HourOfWeek(t time.Time) int {
	t = t.In(Location())
	return (int(t.Weekday())+6)%7*24 + t.Hour()
}

func hourKey(t time.Time) int64 { return t.Unix() / 3600 }

// Record counts one person's turn at t.
func Record(db *store.DB, at time.Time) error {
	_, err := db.Exec(`INSERT INTO activity_hours(hour,turns) VALUES(?,1)
		ON CONFLICT(hour) DO UPDATE SET turns=turns+1`, hourKey(at))
	return err
}

// IsPerson reports whether a user-turn message is the person typing, not
// an agent's message delivered as a turn (swarm DMs).
func IsPerson(msg string) bool {
	msg = strings.TrimSpace(msg)
	return !strings.HasPrefix(msg, "DM from ") && !strings.HasPrefix(msg, "Cross-swarm DM")
}

// Prune drops hours older than 8 weeks.
func Prune(db *store.DB, now time.Time) error {
	_, err := db.Exec(`DELETE FROM activity_hours WHERE hour < ?`, hourKey(now.Add(-keep)))
	return err
}

// hookTurn is a jcode user turn as the recall hook logs it (run/main.go).
var hookTurn = regexp.MustCompile(`^(\S+) jcode(?:\[[^\]]*\])? \S+: msg="(.*?)" hits=\d+ `)

// SeedOnce counts the person's turns that hook.log already holds from before
// this store recorded any, once: a machine upgraded today has weeks of
// rhythm in it. Harness notices, hug's own injections and agent DMs are
// left out. It returns the number of turns added.
func SeedOnce(db *store.DB, hookLog string) (int, error) {
	if db.GetMeta(seededKey) != "" {
		return 0, nil
	}
	counts := map[int64]int{}
	if f, err := os.Open(hookLog); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for sc.Scan() {
			line := sc.Text()
			m := hookTurn.FindStringSubmatch(line)
			if m == nil || strings.Contains(line, ` skipped="harness notice"`) || strings.Contains(line, ` skipped="own injection"`) || !IsPerson(m[2]) {
				continue
			}
			at, err := time.Parse(time.RFC3339, m[1])
			if err != nil {
				continue
			}
			counts[hourKey(at)]++
		}
		f.Close()
	}
	// The transaction takes the write lock first (store's _txlock=immediate):
	// two processes seeding at once count the log once.
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var seeded string
	if tx.QueryRow(`SELECT value FROM meta WHERE key=?`, seededKey).Scan(&seeded) == nil {
		return 0, nil
	}
	var first int64
	tx.QueryRow(`SELECT COALESCE(MIN(hour),0) FROM activity_hours`).Scan(&first)
	n := 0
	for h, c := range counts {
		if first != 0 && h >= first {
			continue // recorded live already
		}
		n += c
		if _, err := tx.Exec(`INSERT INTO activity_hours(hour,turns) VALUES(?,?)
			ON CONFLICT(hour) DO UPDATE SET turns=turns+excluded.turns`, h, c); err != nil {
			return 0, err
		}
	}
	if _, err := tx.Exec(`INSERT INTO meta(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		seededKey, fmt.Sprintf("%d turns from %s", n, hookLog)); err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

// Hist is the hour-of-week histogram at one moment.
type Hist struct {
	Now   time.Time
	First time.Time // the first hour with a turn; zero when none
	// Turns per bucket over the last Weeks weeks.
	Turns [HoursPerWeek]int
	// Seen is how many times each bucket's hour passed since First (at most
	// Weeks); Active how many of those had a turn.
	Seen, Active [HoursPerWeek]int
}

// Read builds the histogram from the stored hours as of now.
func Read(db *store.DB, now time.Time) (*Hist, error) {
	h := &Hist{Now: now}
	var first int64
	if err := db.QueryRow(`SELECT COALESCE(MIN(hour),0) FROM activity_hours`).Scan(&first); err != nil {
		return nil, err
	}
	if first == 0 {
		return h, nil
	}
	h.First = time.Unix(first*3600, 0)
	from := max(first, hourKey(now.Add(-Weeks*7*24*time.Hour)))
	rows, err := db.Query(`SELECT hour, turns FROM activity_hours WHERE hour >= ?`, from)
	if err != nil {
		return nil, err
	}
	turns := map[int64]int{}
	for rows.Next() {
		var k int64
		var c int
		if rows.Scan(&k, &c) == nil {
			turns[k] = c
		}
	}
	rows.Close()
	for k := from; k < hourKey(now); k++ {
		b := HourOfWeek(time.Unix(k*3600, 0))
		h.Seen[b]++
		if c := turns[k]; c > 0 {
			h.Active[b]++
			h.Turns[b] += c
		}
	}
	return h, nil
}

// Ready reports whether a week of activity exists.
func (h *Hist) Ready() bool {
	return !h.First.IsZero() && h.Now.Sub(h.First) >= MinDays*24*time.Hour
}

// Learning is the line shown while the histogram is not ready.
func (h *Hist) Learning() string {
	days := 0
	if !h.First.IsZero() {
		days = int(h.Now.Sub(h.First) / (24 * time.Hour))
	}
	return fmt.Sprintf("learning your week: %d of %d days of activity", min(days, MinDays), MinDays)
}

// Quiet reports whether bucket b is quiet: a turn in it in at most a quarter
// of the weeks seen. One late night in a month does not break a stretch; in
// the only week there is, it does.
func (h *Hist) Quiet(b int) bool {
	return h.Seen[b] > 0 && h.Active[b]*4 <= h.Seen[b]
}

// Stretch is a run of quiet hours of the week.
type Stretch struct {
	Start int // bucket of its first hour
	Hours int
}

var days = [7]string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}

func (s Stretch) String() string {
	return fmt.Sprintf("%s %02d:00 for %d h", days[s.Start/24], s.Start%24, s.Hours)
}

// stretches are the maximal quiet runs, around the end of the week; nil
// when every hour is quiet or none is.
func (h *Hist) stretches() []Stretch {
	var out []Stretch
	for b := 0; b < HoursPerWeek; b++ {
		if !h.Quiet(b) || h.Quiet((b+HoursPerWeek-1)%HoursPerWeek) {
			continue
		}
		n := 0
		for n < HoursPerWeek && h.Quiet((b+n)%HoursPerWeek) {
			n++
		}
		out = append(out, Stretch{b, n})
	}
	return out
}

// Longest is the week's longest quiet stretch (the earliest in the week on a
// tie).
func (h *Hist) Longest() (Stretch, bool) {
	var best Stretch
	for _, s := range h.stretches() {
		if s.Hours > best.Hours {
			best = s
		}
	}
	return best, best.Hours > 0
}

// ForDay is the longest quiet stretch that begins on day d (0 Monday).
func (h *Hist) ForDay(d int) (Stretch, bool) {
	var best Stretch
	for _, s := range h.stretches() {
		if s.Start/24 == d && s.Hours > best.Hours {
			best = s
		}
	}
	return best, best.Hours > 0
}

// Window is the dream window at t: the day's quiet stretch running at t, or
// else the next one to begin. Each day's longest stretch is a window, so
// the daily pass has one a day. ok is false before a week of data or when
// the week has no quiet stretch.
func (h *Hist) Window(t time.Time) (start, end time.Time, ok bool) {
	if !h.Ready() {
		return time.Time{}, time.Time{}, false
	}
	length := map[int]int{}
	for d := 0; d < 7; d++ {
		if s, ok := h.ForDay(d); ok {
			length[s.Start] = s.Hours
		}
	}
	if len(length) == 0 {
		return time.Time{}, time.Time{}, false
	}
	now := hourKey(t)
	for k := now - HoursPerWeek; k <= now+HoursPerWeek; k++ {
		at := time.Unix(k*3600, 0)
		n, is := length[HourOfWeek(at)]
		if !is {
			continue
		}
		if e := at.Add(time.Duration(n) * time.Hour); e.After(t) {
			return at.In(t.Location()), e.In(t.Location()), true
		}
	}
	return time.Time{}, time.Time{}, false
}

// Grid is the histogram as text: a row per day, turns per hour, "." for a
// quiet hour with no turn at all.
func (h *Hist) Grid() string {
	var b strings.Builder
	b.WriteString("    ")
	for hr := 0; hr < 24; hr++ {
		fmt.Fprintf(&b, " %3d", hr)
	}
	b.WriteString("\n")
	for d := 0; d < 7; d++ {
		b.WriteString(days[d])
		for hr := 0; hr < 24; hr++ {
			bk := d*24 + hr
			switch {
			case h.Turns[bk] == 0:
				b.WriteString("   .")
			case h.Quiet(bk):
				fmt.Fprintf(&b, " %2d~", h.Turns[bk])
			default:
				fmt.Fprintf(&b, " %3d", h.Turns[bk])
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("turns per hour of the week, last " + fmt.Sprint(Weeks) + " weeks; . none, ~ quiet (a turn in at most a quarter of the weeks)\n")
	return b.String()
}
